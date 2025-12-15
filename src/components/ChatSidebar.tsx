import { useState, Fragment } from 'react';
import clsx from 'clsx';
import { MessageSquare, CopyIcon, RefreshCcwIcon } from 'lucide-react';
import { useAgentChat } from '@/lib/hooks/useAgentChat';
import { ButtonGroup } from '@/components/ui/button-group';
import { Button } from '@/components/ui/button';
import type { TransportType } from '@/lib/ag-ui/ag-ui-client';
import {
  Conversation,
  ConversationContent,
  ConversationEmptyState,
  ConversationScrollButton,
} from '@/components/ai-elements/conversation';
import {
  Message,
  MessageContent,
  MessageResponse,
  MessageActions,
  MessageAction,
} from '@/components/ai-elements/message';
import { extractMessageContent, extractMessageRole, isUserMessage } from '@/lib/ag-ui/message-helpers';
import type { Message as AgUIMessage } from '@ag-ui/core';

export function ChatSidebar() {
  const [input, setInput] = useState<string>('');
  const [transport, setTransport] = useState<TransportType>('connectrpc');
  
  const agentUrl = transport === 'http-sse' ? '/api/agent/sse' : '/api/agent';
  
  const { messages, isLoading, isConnected, sendMessage, clearMessages } = useAgentChat({
    agentUrl,
    threadId: 'chat-thread',
    transport,
  });

  const handleSubmit = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    if (!input.trim() || isLoading) return;

    await sendMessage(input);
    setInput('');
  };

  return (
    <div 
      className={clsx(
        'fixed right-0 top-0 w-[400px] h-screen bg-white',
        'shadow-[-2px_0_10px_rgba(0,0,0,0.1)] flex flex-col',
        'transition-transform duration-300 ease-in-out z-[1000]',
        'translate-x-0'
      )}
    >
      <div className="p-5 border-b border-gray-200 bg-gray-50">
        <div className="flex items-center justify-between mb-3">
          <h2 className="m-0 text-xl font-semibold text-gray-800">Sidebar Chatbot Demo</h2>
          {isConnected ? (
            <span className="flex items-center gap-1 text-xs text-green-600">
              <span className="w-2 h-2 bg-green-600 rounded-full"></span>
              Connected
            </span>
          ) : (
            <span className="flex items-center gap-1 text-xs text-gray-500">
              <span className="w-2 h-2 bg-gray-400 rounded-full"></span>
              Connecting...
            </span>
          )}
        </div>
        <div className="flex items-center gap-2">
          <span className="text-xs text-gray-600 font-medium">Transport:</span>
          <ButtonGroup>
            <Button
              variant={transport === 'http-sse' ? 'default' : 'outline'}
              size="sm"
              onClick={() => {
                setTransport('http-sse');
                clearMessages();
              }}
              disabled={isLoading}
            >
              HTTP/SSE
            </Button>
            <Button
              variant={transport === 'connectrpc' ? 'default' : 'outline'}
              size="sm"
              onClick={() => {
                setTransport('connectrpc');
                clearMessages();
              }}
              disabled={isLoading}
            >
              ConnectRPC
            </Button>
          </ButtonGroup>
        </div>
      </div>

      <Conversation className="flex-1" style={{ minHeight: 0 }}>
        <ConversationContent>
          {messages.length === 0 ? (
            <ConversationEmptyState
              icon={<MessageSquare className="size-12" />}
              title={isConnected ? 'Welcome! Start a conversation.' : 'Connecting to agent...'}
              description={
                isConnected
                  ? 'Type a message below to begin chatting.'
                  : 'Establishing connection to the agent...'
              }
            />
          ) : (
            <>
              {messages.map((message: AgUIMessage, messageIndex) => {
                const role = extractMessageRole(message);
                const content = extractMessageContent(message);
                
                return (
                  <Fragment key={message.id}>
                    <Message from={role}>
                      <MessageContent>
                        <MessageResponse>{content}</MessageResponse>
                      </MessageContent>
                    </Message>
                    {role === 'assistant' && messageIndex === messages.length - 1 && (
                      <MessageActions>
                        <MessageAction
                          onClick={() => navigator.clipboard.writeText(content)}
                          label="Copy"
                        >
                          <CopyIcon className="size-3" />
                        </MessageAction>
                        <MessageAction
                          onClick={() => {
                            // Find the last user message
                            for (let i = messages.length - 2; i >= 0; i--) {
                              if (isUserMessage(messages[i])) {
                                const userContent = extractMessageContent(messages[i]);
                                sendMessage(userContent);
                                break;
                              }
                            }
                          }}
                          label="Retry"
                        >
                          <RefreshCcwIcon className="size-3" />
                        </MessageAction>
                      </MessageActions>
                    )}
                  </Fragment>
                );
              })}

              {isLoading && (
                <div className="flex items-start">
                  <div className="max-w-[80%]">
                    <div className="px-4 py-3 bg-gray-100 text-gray-800 rounded-[18px_18px_18px_4px] shadow-sm">
                      <div className="flex gap-1">
                        <span className="animate-bounce">.</span>
                        <span className="animate-bounce delay-100">.</span>
                        <span className="animate-bounce delay-200">.</span>
                      </div>
                    </div>
                  </div>
                </div>
              )}
            </>
          )}
        </ConversationContent>
        <ConversationScrollButton />
      </Conversation>

      <form className="p-5 border-t border-gray-200 flex gap-2.5 bg-gray-50" onSubmit={handleSubmit}>
        <input
          type="text"
          value={input}
          onChange={(e) => setInput(e.target.value)}
          placeholder="Type your message..."
          className={clsx(
            'flex-1 px-4 py-3 border border-gray-200 rounded-full text-sm outline-none',
            'focus:border-blue-600 transition-colors',
            'disabled:bg-gray-100 disabled:cursor-not-allowed'
          )}
          disabled={isLoading}
        />
        <button
          type="submit"
          className={clsx(
            'px-6 py-3 bg-blue-600 text-white border-none rounded-full',
            'text-sm font-medium cursor-pointer transition-colors',
            'hover:bg-blue-700',
            'disabled:bg-gray-300 disabled:cursor-not-allowed'
          )}
          disabled={!input.trim() || isLoading}
        >
          Send
        </button>
      </form>
    </div>
  );
}