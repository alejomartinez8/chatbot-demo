/**
 * Message helpers for AG-UI protocol
 * Provides utilities to extract information from AG-UI Message types for UI components
 */
import type { Message as AgUIMessage } from '@ag-ui/core';

/**
 * Extract text content from an AG-UI Message
 * Handles both string content and array of parts
 */
export function extractMessageContent(message: AgUIMessage): string {
  if (!message.content) {
    return '';
  }
  
  if (typeof message.content === 'string') {
    return message.content;
  }
  
  if (Array.isArray(message.content)) {
    // Extract text from parts
    return message.content
      .map((part) => {
        if (typeof part === 'string') {
          return part;
        } else if (part && typeof part === 'object' && 'text' in part) {
          return String(part.text);
        }
        return '';
      })
      .join('');
  }
  
  return '';
}

/**
 * Extract role from AG-UI Message for UI components
 * Maps AG-UI roles to UI-compatible roles
 */
export function extractMessageRole(message: AgUIMessage): 'user' | 'assistant' | 'system' | 'data' {
  switch (message.role) {
    case 'user':
      return 'user';
    case 'assistant':
      return 'assistant';
    case 'system':
      return 'system';
    case 'tool':
    case 'developer':
    default:
      return 'data';
  }
}

/**
 * Check if a message is a user message
 */
export function isUserMessage(message: AgUIMessage): boolean {
  return message.role === 'user';
}

/**
 * Check if a message is an assistant message
 */
export function isAssistantMessage(message: AgUIMessage): boolean {
  return message.role === 'assistant';
}

