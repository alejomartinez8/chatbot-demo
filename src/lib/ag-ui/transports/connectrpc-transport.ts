/**
 * ConnectRPC Transport Implementation
 * Uses ConnectRPC for type-safe AG-UI communication.
 */
import type { RunAgentInput as AgUIRunAgentInput } from '@ag-ui/core';
import { EventSchemas } from '@ag-ui/core';
import { createClient } from '@connectrpc/connect';
import { createConnectTransport } from '@connectrpc/connect-web';
import { AGUIService } from '@/gen/agui/v1/agui_pb';
import type { RunAgentInput, AGUIEvent, Message, Tool, Context } from '@/gen/agui/v1/agui_pb';
import type { Value } from '@bufbuild/protobuf/wkt';
import { BaseTransport, type TransportConfig } from './base-transport';
import type { AgUIEvent } from '../types/ag-ui-events';

export class ConnectRpcTransport extends BaseTransport {
  private client: ReturnType<typeof createClient<typeof AGUIService>>;
  private abortController: AbortController | null = null;

  constructor(config: TransportConfig) {
    super(config);
    
    const transport = createConnectTransport({
      baseUrl: config.url,
      ...(config.headers && { fetch: (input, init) => {
        return fetch(input, {
          ...init,
          headers: {
            ...init?.headers,
            ...config.headers,
          },
        });
      }}),
    });

    this.client = createClient(AGUIService, transport);
  }

  /**
   * Connect to the agent via ConnectRPC.
   */
  async connect(): Promise<void> {
    if (this.status === 'connected' || this.status === 'connecting') {
      return;
    }

    this.updateStatus('connecting');

    try {
      // Create AbortController for connection management
      this.abortController = new AbortController();

      // Standard ag-ui RunAgentInput format - empty initial connection
      const threadId = crypto.randomUUID();
      const runId = crypto.randomUUID();

      const body: AgUIRunAgentInput = {
        threadId,
        runId,
        state: {},
        messages: [],
        tools: [],
        context: [],
        forwardedProps: {},
      };

      const protoInput = this.convertRunAgentInput(body);
      const stream = this.client.runAgent(protoInput, {
        signal: this.abortController.signal,
      });

      this.readStream(stream as AsyncIterable<AGUIEvent>);

      this.updateStatus('connected');
    } catch (error) {
      this.handleError(error as Error);
      throw error;
    }
  }

  /**
   * Disconnect from the agent.
   */
  disconnect(): void {
    if (this.abortController) {
      this.abortController.abort();
      this.abortController = null;
    }
    this.updateStatus('disconnected');
  }

  /**
   * Send a message to the agent.
   */
  async send(runAgentInput: AgUIRunAgentInput): Promise<void> {
    if (this.status !== 'connected') {
      throw new Error('Not connected to agent');
    }

    if (this.abortController) {
      this.abortController.abort();
    }

    this.abortController = new AbortController();

    try {
      const protoInput = this.convertRunAgentInput(runAgentInput);
      const stream = this.client.runAgent(protoInput, {
        signal: this.abortController.signal,
      });

      this.readStream(stream as AsyncIterable<AGUIEvent>);
    } catch (error) {
      this.handleError(error as Error);
      throw error;
    }
  }

  /**
   * Read and process a stream of AGUIEvent.
   */
  private async readStream(stream: AsyncIterable<AGUIEvent>): Promise<void> {
    try {
      for await (const protoEvent of stream) {
        const aguiEvent = this.convertAGUIEvent(protoEvent);
        if (aguiEvent) {
          this.emitEvent({
            id: crypto.randomUUID(),
            event: aguiEvent,
            timestamp: Date.now(),
            direction: 'incoming',
          });
        }
      }
    } catch (error) {
      if (!(error instanceof Error && error.name === 'AbortError')) {
        this.handleError(error as Error);
      }
    }
  }

  /**
   * Convert AG-UI RunAgentInput to protobuf RunAgentInput.
   */
  private convertRunAgentInput(input: AgUIRunAgentInput): RunAgentInput {
    const messages: Message[] = (input.messages || []).map((msg) => {
      const protoMsg: Message = {
        id: msg.id,
        role: msg.role,
        name: ('name' in msg && msg.name) ? msg.name : '',
      } as Message;

      if ('content' in msg && msg.content !== undefined) {
        protoMsg.content = this.convertToValue(msg.content);
      }

      if ('toolCalls' in msg && msg.toolCalls !== undefined) {
        protoMsg.toolCalls = this.convertToValue(msg.toolCalls);
      }

      return protoMsg;
    });

    const tools: Tool[] = (input.tools || []).map((tool: any) => {
      return {
        name: tool.name || '',
        description: tool.description || '',
        parameters: tool.parameters || {},
      } as Tool;
    });

    const context: Context[] = (input.context || []).map((ctx: any) => {
      return {
        description: ctx.description || '',
        value: ctx.value || '',
      } as Context;
    });

    return {
      threadId: input.threadId,
      runId: input.runId,
      parentRunId: input.parentRunId || '',
      state: input.state || {},
      messages,
      tools,
      context,
      forwardedProps: input.forwardedProps || {},
    } as RunAgentInput;
  }

  /**
   * Convert a JavaScript value to protobuf Value.
   */
  private convertToValue(value: any): Value {
    if (value === null || value === undefined) {
      return {
        $typeName: 'google.protobuf.Value',
        kind: { case: 'nullValue', value: 0 },
      } as Value;
    }
    if (typeof value === 'string') {
      return {
        $typeName: 'google.protobuf.Value',
        kind: { case: 'stringValue', value },
      } as Value;
    }
    if (typeof value === 'number') {
      return {
        $typeName: 'google.protobuf.Value',
        kind: { case: 'numberValue', value },
      } as Value;
    }
    if (typeof value === 'boolean') {
      return {
        $typeName: 'google.protobuf.Value',
        kind: { case: 'boolValue', value },
      } as Value;
    }
    if (Array.isArray(value)) {
      return {
        $typeName: 'google.protobuf.Value',
        kind: {
          case: 'listValue',
          value: {
            values: value.map((v) => this.convertToValue(v)),
          },
        },
      } as Value;
    }
    if (typeof value === 'object') {
      return {
        $typeName: 'google.protobuf.Value',
        kind: {
          case: 'structValue',
          value: {
            fields: Object.fromEntries(
              Object.entries(value).map(([k, v]) => [k, this.convertToValue(v)])
            ),
          },
        },
      } as Value;
    }
    return {
      $typeName: 'google.protobuf.Value',
      kind: { case: 'stringValue', value: String(value) },
    } as Value;
  }

  /**
   * Convert protobuf AGUIEvent to AG-UI event.
   */
  private convertAGUIEvent(protoEvent: AGUIEvent): AgUIEvent | null {
    try {
      const eventType = protoEvent.type;
      const eventData = protoEvent.data || {};
      const event = {
        type: eventType,
        ...eventData,
      };

      const result = EventSchemas.safeParse(event);

      if (result.success) {
        return result.data as AgUIEvent;
      }

      const isUnsupportedEvent = result.error.errors.some(
        (err) => err.code === 'invalid_union_discriminator' && err.path.length === 1 && err.path[0] === 'type'
      );

      this.emitEvent({
        id: crypto.randomUUID(),
        event: event as AgUIEvent,
        timestamp: Date.now(),
        direction: 'incoming',
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
      });

      return event as AgUIEvent;
    } catch (error) {
      this.config.onError?.(error as Error);
      return null;
    }
  }
}

