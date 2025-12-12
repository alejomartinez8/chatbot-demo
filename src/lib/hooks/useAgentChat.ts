import { useRef, useEffect, useState, useCallback } from 'react';
import type { RunAgentInput } from '@ag-ui/core';
import { AgUIClient } from '@/lib/ag-ui/ag-ui-client';
import type { TransportEvent } from '@/lib/ag-ui/types/ag-ui-events';
import { Message } from '@/lib/types';

interface UseAgentChatOptions {
  agentUrl: string;
  threadId?: string;
}

interface UseAgentChatReturn {
  messages: Message[];
  isLoading: boolean;
  error: string | null;
  isConnected: boolean;
  sendMessage: (content: string) => Promise<void>;
  clearMessages: () => void;
}

export function useAgentChat({
  agentUrl,
  threadId = 'default-thread',
}: UseAgentChatOptions): UseAgentChatReturn {
  const [messages, setMessages] = useState<Message[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [isConnected, setIsConnected] = useState(false);
  const clientRef = useRef<AgUIClient | null>(null);
  const assistantBufferRef = useRef('');
  const assistantMessageIdRef = useRef<string | null>(null);

  const handleEvent = useCallback((transportEvent: TransportEvent) => {
    const event = transportEvent.event as { type?: string; delta?: string };

    switch (event.type) {
      case 'TEXT_MESSAGE_START': {
        assistantBufferRef.current = '';
        assistantMessageIdRef.current = crypto.randomUUID();
        break;
      }
      case 'TEXT_MESSAGE_CONTENT': {
        if (typeof event.delta === 'string') {
          assistantBufferRef.current += event.delta;
        }
        break;
      }
      case 'TEXT_MESSAGE_END': {
        const content = assistantBufferRef.current.trim();
        if (content) {
          const assistantMessage: Message = {
            id: assistantMessageIdRef.current || crypto.randomUUID(),
            role: 'assistant',
            content,
          };
          setMessages((prev) => [...prev, assistantMessage]);
        }
        assistantBufferRef.current = '';
        assistantMessageIdRef.current = null;
        setIsLoading(false);
        break;
      }
      default:
        break;
    }
  }, []);

  // Initialize the AgUI client and connect
  useEffect(() => {
    const client = new AgUIClient({
      url: agentUrl,
      onEvent: handleEvent,
      onStateChange: (state) => {
        setIsConnected(state === 'connected');
      },
      onError: (err) => {
        setError(err.message);
      },
    });

    clientRef.current = client;

    const connectClient = async () => {
      setError(null);

      try {
        await client.connect();
      } catch (err) {
        const errorMessage = 'Failed to connect to agent. Make sure it is running and reachable.';
        setError(errorMessage);
        console.error('Agent connection error:', err);
      }
    };

    connectClient();

    return () => {
      client.disconnect();
      clientRef.current = null;
    };
  }, [agentUrl, handleEvent]);

  const sendMessage = useCallback(async (content: string) => {
    if (!content.trim() || isLoading || !clientRef.current) return;

    const userMessage: Message = {
      id: crypto.randomUUID(),
      role: 'user',
      content,
    };

    setMessages((prev) => [...prev, userMessage]);
    setIsLoading(true);
    setError(null);

    try {
      // Prepare the conversation messages
      const conversationMessages: Array<{
        id: string;
        role: 'user' | 'assistant';
        content: string;
      }> = [
        ...messages.map(m => ({
          id: m.id,
          role: m.role,
          content: m.content,
        })),
        {
          id: userMessage.id,
          role: 'user' as const,
          content,
        }
      ];

      // Build RunAgentInput payload
      const runInput: RunAgentInput = {
        threadId,
        runId: crypto.randomUUID(),
        messages: conversationMessages as any, // AG-UI core Message type is complex; cast for now
        state: {},
        tools: [],
        context: [],
        forwardedProps: {},
      };

      // Ensure the client is connected before sending
      if (clientRef.current.getStatus() !== 'connected') {
        await clientRef.current.connect();
      }

      await clientRef.current.sendMessage(runInput);
      
      // Note: isLoading will be set to false by TEXT_MESSAGE_END event
    } catch (err) {
      console.error('Error calling agent:', err);
      
      const errorMessage = `Failed to send message. Check that the agent is running at ${agentUrl}`;
      setError(errorMessage);
      
      // Add error message to chat
      const errorChatMessage: Message = {
        id: crypto.randomUUID(),
        role: 'assistant',
        content: errorMessage,
      };
      
      setMessages((prev) => [...prev, errorChatMessage]);
      setIsLoading(false);
    }
  }, [messages, isLoading, threadId, agentUrl]);

  const clearMessages = useCallback(() => {
    setMessages([]);
    setError(null);
  }, []);

  return {
    messages,
    isLoading,
    error,
    isConnected,
    sendMessage,
    clearMessages,
  };
}