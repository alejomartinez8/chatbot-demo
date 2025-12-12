/**
 * HTTP/SSE Transport Implementation
 * Uses HTTP POST with Server-Sent Events for AG-UI communication.
 */
import type { RunAgentInput } from '@ag-ui/core';
import { EventSchemas } from '@ag-ui/core';
import { BaseTransport, type TransportConfig } from './base-transport';
import type { AgUIEvent } from '../types/ag-ui-events';

export class HttpSseTransport extends BaseTransport {
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 5;
  private reconnectDelay = 1000; // ms
  private activeStreams: Set<ReadableStreamDefaultReader<Uint8Array>> = new Set();
  private activeResponses: Set<Response> = new Set();
  private currentAbortController: AbortController | null = null;

  constructor(config: TransportConfig) {
    super(config);
  }

  /**
   * Connect to the agent via HTTP/SSE.
   */
  async connect(): Promise<void> {
    if (this.status === 'connected' || this.status === 'connecting') {
      return;
    }

    this.updateStatus('connecting');

    try {
      // Create AbortController for initial connection
      this.currentAbortController = new AbortController();

      // Standard ag-ui RunAgentInput format - empty initial connection
      const threadId = crypto.randomUUID();
      const runId = crypto.randomUUID();

      const body: RunAgentInput = {
        threadId,
        runId,
        state: {},
        messages: [],
        tools: [],
        context: [],
        forwardedProps: {},
      };

      const response = await fetch(this.config.url, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Accept: 'text/event-stream',
          ...this.config.headers,
        },
        body: JSON.stringify(body),
        signal: this.currentAbortController.signal,
      });

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      if (!response.body) {
        throw new Error('Response body is null');
      }

      this.updateStatus('connected');
      this.reconnectAttempts = 0;

      // Keep reference to response to prevent garbage collection
      this.activeResponses.add(response);

      // Read SSE stream in the background
      this.readStream(response.body, response);
    } catch (error) {
      this.handleError(error as Error);
      throw error;
    }
  }

  /**
   * Disconnect from the agent.
   */
  disconnect(): void {
    // Abort current request
    if (this.currentAbortController) {
      this.currentAbortController.abort();
      this.currentAbortController = null;
    }

    // Cancel all active streams
    for (const reader of this.activeStreams) {
      reader.cancel().catch(() => {
        /* ignore errors */
      });
    }
    this.activeStreams.clear();
    this.activeResponses.clear();
    this.updateStatus('disconnected');
    this.reconnectAttempts = 0;
  }

  /**
   * Send a message to the agent.
   */
  async send(runAgentInput: RunAgentInput): Promise<void> {
    if (this.status !== 'connected') {
      throw new Error('Not connected to agent');
    }

    // Cancel previous request if active
    if (this.currentAbortController) {
      this.currentAbortController.abort();
    }

    // Create new AbortController for this request
    this.currentAbortController = new AbortController();

    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
      Accept: 'text/event-stream',
      ...this.config.headers,
    };

    // Add x-thread-id header if threadId is provided
    if (runAgentInput.threadId) {
      headers['x-thread-id'] = runAgentInput.threadId;
    }

    const response = await fetch(this.config.url, {
      method: 'POST',
      headers,
      body: JSON.stringify(runAgentInput),
      signal: this.currentAbortController.signal,
    });

    if (!response.ok) {
      const errorText = await response.text();
      throw new Error(`HTTP error! status: ${response.status} - ${errorText}`);
    }

    if (!response.body) {
      throw new Error('Response body is null');
    }

    // Keep reference to response to prevent garbage collection
    this.activeResponses.add(response);

    // Start reading the new stream in the background (don't await)
    this.readStream(response.body, response);
  }

  /**
   * Attempt reconnection with backoff.
   */
  private async attemptReconnect(): Promise<void> {
    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      this.updateStatus('error');
      return;
    }

    this.reconnectAttempts += 1;
    this.updateStatus('reconnecting');

    await new Promise((resolve) => setTimeout(resolve, this.reconnectDelay));
    await this.connect();
  }

  /**
   * Read and process an SSE stream.
   */
  private async readStream(body: ReadableStream<Uint8Array>, response: Response): Promise<void> {
    const reader = body.getReader();
    this.activeStreams.add(reader);
    const decoder = new TextDecoder();
    let buffer = '';

    try {
      while (true) {
        const { done, value } = await reader.read();

        if (done) {
          break;
        }

        if (!value) {
          continue;
        }

        const chunk = decoder.decode(value, { stream: true });
        buffer += chunk;
        const lines = buffer.split(/\r?\n/); // Handle both \n and \r\n
        buffer = lines.pop() || '';

        for (const line of lines) {
          const trimmedLine = line.trim();

          if (trimmedLine.startsWith('data: ')) {
            const data = trimmedLine.slice(6);
            if (data === '[DONE]') {
              continue;
            }
            this.handleMessage(data);
          }
        }
      }
    } catch (error) {
      // Ignore AbortError - it's expected when we cancel a stream
      if (!(error instanceof Error && error.name === 'AbortError')) {
        this.handleError(error as Error);
        await this.attemptReconnect();
      }
    } finally {
      this.activeStreams.delete(reader);
      this.activeResponses.delete(response);
    }
  }

  /**
   * Handle incoming message.
   */
  private handleMessage(data: string): void {
    try {
      const parsed = JSON.parse(data);
      const { event: validated, validationError } = this.validateEvent(parsed);

      if (validated) {
        this.emitEvent({
          id: crypto.randomUUID(),
          event: validated,
          timestamp: Date.now(),
          direction: 'incoming',
          raw: data,
          validationError,
        });
      }
    } catch (error) {
      this.config.onError?.(error as Error);
    }
  }

  /**
   * Validate event against ag-ui schema.
   */
  private validateEvent(data: unknown): {
    event: AgUIEvent | null;
    validationError?: {
      message: string;
      errors: Array<{ path: string; message: string }>;
      isUnsupportedEvent?: boolean;
    };
  } {
    try {
      const result = EventSchemas.safeParse(data);

      if (result.success) {
        return { event: result.data as AgUIEvent };
      }

      // Check if this is an unsupported event type (discriminator error on 'type' field)
      const isUnsupportedEvent = result.error.errors.some(
        (err) => err.code === 'invalid_union_discriminator' && err.path.length === 1 && err.path[0] === 'type'
      );

      return {
        event: data as AgUIEvent,
        validationError: {
          message: isUnsupportedEvent
            ? 'Unsupported event type (not in AG-UI protocol)'
            : 'Event validation failed (invalid fields)',
          errors: result.error.errors.map((err) => ({
            path: err.path.join('.'),
            message: err.message,
          })),
          isUnsupportedEvent,
        },
      };
    } catch (error) {
      return {
        event: data as AgUIEvent,
        validationError: {
          message: 'Event validation exception',
          errors: [{ path: '', message: String(error) }],
        },
      };
    }
  }
}


