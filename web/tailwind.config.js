/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  theme: {
    extend: {
      colors: {
        nodeTrigger: '#6366f1',
        nodeLLM:     '#9333ea',
        nodeComm:    '#16a34a',
        nodeData:    '#ea580c',
        nodeControl: '#eab308',
        nodeCode:    '#6b7280',
      },
    },
  },
  plugins: [],
};
