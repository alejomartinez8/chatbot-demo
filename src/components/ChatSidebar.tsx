import { useState } from 'react';
import clsx from 'clsx';
import { MessageSquare } from 'lucide-react';
import { useAgentChat } from '@/lib/hooks/useAgentChat';
import {
  Conversation,
  ConversationContent,
  ConversationEmptyState,
  ConversationScrollButton,
} from '@/components/ai-elements/conversation';

export function ChatSidebar() {
  const [isOpen, setIsOpen] = useState<boolean>(true);
  const [input, setInput] = useState<string>('');
  
  // Use the custom hook to manage agent communication
  const { messages, isLoading, isConnected, isConnecting, sendMessage } = useAgentChat({
    agentUrl: '/api/agent/',
    threadId: 'chat-thread',
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
        isOpen ? 'translate-x-0' : 'translate-x-full'
      )}
    >
      <div className="p-5 border-b border-gray-200 flex justify-between items-center bg-gray-50">
        <div className="flex items-center gap-2">
          <h2 className="m-0 text-xl font-semibold text-gray-800">Sidebar Chatbot Demo</h2>
          {isConnecting && (
            <span className="text-xs text-gray-500">Connecting...</span>
          )}
          {!isConnecting && isConnected && (
            <span className="flex items-center gap-1 text-xs text-green-600">
              <span className="w-2 h-2 bg-green-600 rounded-full"></span>
              Connected
            </span>
          )}
          {!isConnecting && !isConnected && (
            <span className="flex items-center gap-1 text-xs text-red-600">
              <span className="w-2 h-2 bg-red-600 rounded-full"></span>
              Disconnected
            </span>
          )}
        </div>
        <button 
          className={clsx(
            'bg-transparent border-none text-2xl cursor-pointer',
            'text-gray-600 px-2.5 py-1 rounded',
            'hover:bg-gray-200 transition-colors'
          )}
          onClick={() => setIsOpen(!isOpen)}
          aria-label="Toggle sidebar"
        >
          {isOpen ? '−' : '+'}
        </button>
      </div>

      {isOpen && (
        <>
          <Conversation className="flex-1" style={{ minHeight: 0 }}>
            <ConversationContent>
              {messages.length === 0 ? (
                <ConversationEmptyState
                  icon={<MessageSquare className="size-12" />}
                  title={
                    isConnecting
                      ? 'Connecting to agent...'
                      : !isConnected
                      ? '⚠️ Agent not connected'
                      : 'Welcome! Start a conversation.'
                  }
                  description={
                    !isConnecting && !isConnected
                      ? 'Make sure the agent is running on localhost:8000'
                      : 'Type a message below to begin chatting.'
                  }
                />
              ) : (
                <>
                  {messages.map((message) => (
                    <div
                      key={message.id}
                      className={clsx(
                        'flex flex-col',
                        message.role === 'user' ? 'items-end' : 'items-start'
                      )}
                    >
                      <div className="max-w-[80%]">
                        <div
                          className={clsx(
                            'px-4 py-3 break-words',
                            message.role === 'user'
                              ? 'bg-blue-600 text-white rounded-[18px_18px_4px_18px] shadow-md'
                              : 'bg-gray-100 text-gray-800 rounded-[18px_18px_18px_4px] shadow-sm'
                          )}
                        >
                          {message.content}
                        </div>
                      </div>
                    </div>
                  ))}

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
        </>
      )}
    </div>
  );
}
