import { useState } from 'react'
import clsx from 'clsx'

export function ChatSidebar() {
  const [isOpen, setIsOpen] = useState<boolean>(true)
  const [input, setInput] = useState<string>('')
  const [messages, setMessages] = useState<Array<{ id: string; role: 'user' | 'assistant'; content: string }>>([])

  const handleSubmit = (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault()
    if (!input.trim()) return

    const newMessage = {
      id: crypto.randomUUID(),
      role: 'user' as const,
      content: input,
    }

    setMessages([...messages, newMessage])
    setInput('')
  }

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
        <h2 className="m-0 text-xl font-semibold text-gray-800">Chatbot Demo</h2>
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
          <div className="flex-1 overflow-y-auto p-5 flex flex-col gap-4">
            {messages.length === 0 && (
              <div className="text-center text-gray-600 py-10 px-5">
                <p>Welcome! Start a conversation with the chatbot.</p>
              </div>
            )}
            
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
          </div>

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
            />
            <button
              type="submit"
              className={clsx(
                'px-6 py-3 bg-blue-600 text-white border-none rounded-full',
                'text-sm font-medium cursor-pointer transition-colors',
                'hover:bg-blue-700',
                'disabled:bg-gray-300 disabled:cursor-not-allowed'
              )}
              disabled={!input.trim()}
            >
              Send
            </button>
          </form>
        </>
      )}
    </div>
  )
}
