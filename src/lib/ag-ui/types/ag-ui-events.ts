/**
 * AG-UI Event Types
 * Minimal wrapper around @ag-ui/core schemas for inspector-like events.
 */
import { EventSchemas, EventType } from '@ag-ui/core';

export type AgUIEvent = ReturnType<typeof EventSchemas['parse']>;

export interface InspectorEvent {
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


