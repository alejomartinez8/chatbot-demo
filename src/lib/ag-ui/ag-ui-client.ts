/**
 * Thin transport-agnostic AG-UI client. Currently wires HTTP/SSE by default,
 * but the surface stays minimal so additional transports can be slotted in later.
 */
import type { RunAgentInput } from '@ag-ui/core';
import { HttpSseTransport } from './transports/http-sse-transport';
import { ConnectRpcTransport } from './transports/connectrpc-transport';
import { BaseTransport, type ConnectionStatus, type TransportConfig } from './transports/base-transport';

export type { ConnectionStatus, TransportConfig };

export type TransportType = 'http-sse' | 'connectrpc';

export interface AgUIClientConfig extends TransportConfig {
  transport?: TransportType | BaseTransport;
}

/**
 * AG-UI client (defaults to HTTP/SSE). Create once per agent URL and reuse for sends.
 * Swap the transport in the constructor when new transport types are added.
 */
export class AgUIClient {
  private transport: BaseTransport;

  constructor(config: AgUIClientConfig) {
    if (config.transport instanceof BaseTransport) {
      this.transport = config.transport;
    } else if (config.transport === 'connectrpc') {
      this.transport = new ConnectRpcTransport(config);
    } else {
      this.transport = new HttpSseTransport(config);
    }
  }

  /**
   * Connect to the agent.
   */
  async connect(): Promise<void> {
    return this.transport.connect();
  }

  /**
   * Disconnect from the agent.
   */
  disconnect(): void {
    this.transport.disconnect();
  }

  /**
   * Send a message to the agent.
   */
  async sendMessage(runAgentInput: RunAgentInput): Promise<void> {
    return this.transport.send(runAgentInput);
  }

  /**
   * Get current connection status.
   */
  getStatus(): ConnectionStatus {
    return this.transport.getStatus();
  }

  /**
   * Get reconnection info (if transport supports it).
   */
  getReconnectionInfo(): { attempts: number; maxAttempts: number } | null {
    return this.transport.getReconnectionInfo();
  }
}