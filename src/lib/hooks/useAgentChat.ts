import { useRef, useEffect, useState, useCallback } from 'react';
import type { RunAgentInput } from '@ag-ui/core';
import { AgUIClient } from '@/lib/ag-ui/ag-ui-client';
import type { InspectorEvent } from '@/lib/ag-ui/types/ag-ui-events';
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
  isConnecting: boolean;
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
  const [isConnecting, setIsConnecting] = useState(true);
  const clientRef = useRef<AgUIClient | null>(null);
  const assistantBufferRef = useRef('');
  const assistantMessageIdRef = useRef<string | null>(null);

  const handleEvent = useCallback((inspectorEvent: InspectorEvent) => {
    const event = inspectorEvent.event as { type?: string; delta?: string };

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
        if (state === 'connected') {
          setIsConnected(true);
        }
        if (state === 'connecting') {
          setIsConnecting(true);
        }
        if (state === 'disconnected') {
          setIsConnected(false);
          setIsConnecting(false);
        }
        if (state === 'error') {
          setIsConnected(false);
          setIsConnecting(false);
        }
      },
      onError: (err) => {
        setError(err.message);
        setIsLoading(false);
      },
    });

    clientRef.current = client;

    const connectClient = async () => {
      setIsConnecting(true);
      setError(null);

      try {
        await client.connect();
        setIsConnected(true);
      } catch (err) {
        setIsConnected(false);
        const errorMessage = 'Failed to connect to agent. Make sure it is running and reachable.';
        setError(errorMessage);
        console.error('Agent connection error:', err);
      } finally {
        setIsConnecting(false);
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
      // Prepare the conversation messages with proper structure
      const conversationMessages = [
        ...messages.map(m => ({
          id: m.id,
          role: m.role,
          content: m.content,
        })),
        {
          id: userMessage.id,
          role: 'user',
          content,
        }
      ];

      // Call the agent with proper payload structure
      const runInput: RunAgentInput = {
        threadId,
        runId: crypto.randomUUID(),
        messages: conversationMessages as any,
        state: {} as any,
        tools: [],
        context: [],
        forwardedProps: {},
      };

      // Ensure the client is connected before sending
      if (clientRef.current.getStatus() !== 'connected') {
        await clientRef.current.connect();
      }

      await clientRef.current.sendMessage(runInput);
    } catch (err) {
      console.error('Error calling agent:', err);
      
      const errorMessage = 'Sorry, I encountered an error. Please make sure the agent is running on localhost:8000.';
      setError(errorMessage);
      
      // Add error message to chat
      const errorChatMessage: Message = {
        id: crypto.randomUUID(),
        role: 'assistant',
        content: errorMessage,
      };
      
      setMessages((prev) => [...prev, errorChatMessage]);
    } finally {
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
    isConnecting,
    sendMessage,
    clearMessages,
  };
}


