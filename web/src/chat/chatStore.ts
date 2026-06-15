import { create } from 'zustand';

export interface ChatMessage {
  id: string;
  role: 'user' | 'assistant' | 'system';
  content: string;
  proposal?: unknown;
  patch?: unknown;
}

interface ChatState {
  messages: ChatMessage[];
  sending: boolean;
  append: (msg: ChatMessage) => void;
  reset: () => void;
  setSending: (s: boolean) => void;
}

export const useChatStore = create<ChatState>((set) => ({
  messages: [],
  sending: false,
  append: (msg) => set((s) => ({ messages: [...s.messages, msg] })),
  reset: () => set({ messages: [] }),
  setSending: (s) => set({ sending: s }),
}));
