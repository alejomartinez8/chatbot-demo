/**
 * AG-UI Event Types
 * Minimal wrapper around @ag-ui/core schemas for transport events.
 */
import { EventSchemas, EventType } from '@ag-ui/core';

export type AgUIEvent = ReturnType<typeof EventSchemas['parse']>;

/**
 * Wrapper for AG-UI events with transport metadata.
 */
export interface TransportEvent {
  id: string; // Generated UUID for tracking
  event: AgUIEvent; // The actual ag-ui event
  timestamp: number; // When we received it (ms)
  direction: 'incoming' | 'outgoing';
  raw?: string; // Original HTTP message
  validationError?: {
    message: string;
    errors: Array<{ path: string; message: string }>;
    isUnsupportedEvent?: boolean; // True if event type is not in AG-UI protocol
  };
}

export { EventType, EventSchemas };


