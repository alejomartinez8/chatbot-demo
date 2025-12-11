// Chat message types
export type Message = {
  id: string;
  role: 'user' | 'assistant';
  content: string;
};
