/**
 * Base Transport Interface
 * Abstract interface for AG-UI transport implementations.
 */
import type { RunAgentInput } from '@ag-ui/core';
import type { TransportEvent } from '../types/ag-ui-events';

export type ConnectionStatus =
  | 'disconnected'
  | 'connecting'
  | 'connected'
  | 'reconnecting'
  | 'error';

export interface TransportConfig {
  url: string;
  headers?: Record<string, string>;
  onEvent?: (event: TransportEvent) => void;
  onStateChange?: (state: ConnectionStatus) => void;
  onError?: (error: Error) => void;
}

/**
 * Abstract base class for transport implementations.
 */
export abstract class BaseTransport {
  protected config: TransportConfig;
  protected status: ConnectionStatus = 'disconnected';

  constructor(config: TransportConfig) {
    this.config = config;
  }

  /**
   * Connect to the agent.
   */
  abstract connect(): Promise<void>;

  /**
   * Disconnect from the agent.
   */
  abstract disconnect(): void;

  /**
   * Send a message to the agent.
   */
  abstract send(input: RunAgentInput): Promise<void>;

  /**
   * Get current connection status.
   */
  getStatus(): ConnectionStatus {
    return this.status;
  }

  /**
   * Get reconnection info (optional, implemented by transports that support reconnection).
   */
  getReconnectionInfo(): { attempts: number; maxAttempts: number } | null {
    return null;
  }

  /**
   * Update connection status and notify listeners.
   */
  protected updateStatus(status: ConnectionStatus): void {
    this.status = status;
    this.config.onStateChange?.(status);
  }

  /**
   * Handle errors and notify listeners.
   */
  protected handleError(error: Error): void {
    this.updateStatus('error');
    this.config.onError?.(error);
  }

  /**
   * Emit event to listeners.
   */
  protected emitEvent(event: TransportEvent): void {
    try {
      this.config.onEvent?.(event);
    } catch (error) {
      console.error('[Transport] Error in onEvent callback:', error);
      throw error;
    }
  }
}