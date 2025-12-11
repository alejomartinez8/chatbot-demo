import { useRef, useEffect, useState, useCallback } from 'react';
import { HttpAgent } from '@ag-ui/client';
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
  threadId = 'default-thread' 
}: UseAgentChatOptions): UseAgentChatReturn {
  const [messages, setMessages] = useState<Message[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [isConnected, setIsConnected] = useState(false);
  const [isConnecting, setIsConnecting] = useState(true);
  const agentRef = useRef<HttpAgent | null>(null);

  // Initialize the HttpAgent and check connection
  useEffect(() => {
    agentRef.current = new HttpAgent({
      url: agentUrl,
    });

    // Perform health check
    const checkConnection = async () => {
      setIsConnecting(true);
      setError(null);
      
      try {
        console.log('Checking agent connection at:', agentUrl);
        // Use OPTIONS request for health check (less intrusive than POST)
        const response = await fetch(agentUrl, {
          method: 'OPTIONS',
          headers: {
            'Accept': 'application/json',
          },
        });
        
        // Accept 200-299 or 405 (Method Not Allowed) as indication server is up
        if (response.ok || response.status === 405) {
          setIsConnected(true);
          console.log('✅ Agent connected successfully');
        } else {
          setIsConnected(false);
          setError(`Agent returned status ${response.status}`);
          console.error('❌ Agent connection failed:', response.status);
        }
      } catch (err) {
        setIsConnected(false);
        const errorMessage = 'Failed to connect to agent. Make sure it\'s running on localhost:8000';
        setError(errorMessage);
        console.error('❌ Agent connection error:', err);
      } finally {
        setIsConnecting(false);
      }
    };

    checkConnection();

    return () => {
      agentRef.current = null;
    };
  }, [agentUrl]);

  const sendMessage = useCallback(async (content: string) => {
    if (!content.trim() || isLoading || !agentRef.current) return;

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
      const response = agentRef.current.run({
        threadId,
        runId: crypto.randomUUID(),
        messages: conversationMessages as any,
        state: {} as any,
        tools: [],
        context: [],
        forwardedProps: {},
      });

      console.log('Response received:', response);

      // Handle Observable response (streaming)
      let assistantContent = '';
      
      // Check if response is an Observable (has subscribe method)
      if (response && typeof response === 'object' && 'subscribe' in response) {
        console.log('Processing Observable response...');
        
        // Create a promise to wait for all events
        await new Promise<void>((resolve, reject) => {
          const subscription = (response as any).subscribe({
            next: (event: any) => {
              console.log('Event:', event);
              
              // Accumulate text content from TEXT_MESSAGE_CONTENT events
              if (event.type === 'TEXT_MESSAGE_CONTENT' && event.delta) {
                assistantContent += event.delta;
              }
            },
            error: (err: any) => {
              console.error('Observable error:', err);
              reject(err);
            },
            complete: () => {
              console.log('Observable completed');
              subscription.unsubscribe();
              resolve();
            }
          });
        });
      } 
      // Handle non-streaming response (fallback)
      else if (response && typeof response === 'object') {
        const responseObj = response as any;
        
        if ('messages' in responseObj && Array.isArray(responseObj.messages)) {
          const lastMessage = responseObj.messages
            .filter((m: any) => m.role === 'assistant' || m.role === 'model')
            .pop();
          
          if (lastMessage) {
            if (typeof lastMessage.content === 'string') {
              assistantContent = lastMessage.content;
            } else if (lastMessage.content?.text) {
              assistantContent = lastMessage.content.text;
            } else if (Array.isArray(lastMessage.content)) {
              const textParts = lastMessage.content
                .filter((part: any) => part.text)
                .map((part: any) => part.text);
              if (textParts.length > 0) {
                assistantContent = textParts.join('\n');
              }
            }
          }
        } else if ('content' in responseObj && responseObj.content) {
          if (typeof responseObj.content === 'string') {
            assistantContent = responseObj.content;
          } else if (typeof responseObj.content === 'object' && responseObj.content !== null) {
            const contentObj = responseObj.content as any;
            assistantContent = contentObj.text || '';
          }
        }
      }

      // Use accumulated content or default message
      if (!assistantContent) {
        assistantContent = 'I processed your request.';
      }

      console.log('Final assistant content:', assistantContent);

      // Add assistant message
      const assistantMessage: Message = {
        id: crypto.randomUUID(),
        role: 'assistant',
        content: assistantContent,
      };

      setMessages((prev) => [...prev, assistantMessage]);
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
  }, [messages, isLoading, threadId]);

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


