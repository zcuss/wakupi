/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{vue,js,ts,jsx,tsx}'],
  darkMode: 'class',
  theme: {
    extend: {
      colors: {
        wa: {
          green: '#00a884',
          'green-dark': '#008f72',
          'green-light': '#d9fdd3',
          panel: '#f0f2f5',
          'panel-dark': '#202c33',
          bg: '#efeae2',
          'bg-dark': '#0b141a',
          chat: '#ffffff',
          'chat-dark': '#111b21',
          hover: '#f5f6f6',
          'hover-dark': '#2a3942',
          border: '#e9edef',
          'border-dark': '#222d34',
          text: '#111b21',
          'text-dark': '#e9edef',
          muted: '#667781',
          'muted-dark': '#8696a0',
          bubble: '#ffffff',
          'bubble-dark': '#202c33',
          'bubble-out': '#d9fdd3',
          'bubble-out-dark': '#005c4b',
        },
      },
      fontFamily: {
        sans: ['"Segoe UI"', 'Helvetica Neue', 'Helvetica', 'Lucida Grande', 'Arial', 'sans-serif'],
      },
    },
  },
  plugins: [],
}
