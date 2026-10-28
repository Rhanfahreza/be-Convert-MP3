/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  darkMode: 'class',
  theme: {
    extend: {
      fontFamily: {
        sans: ['Geist', '-apple-system', 'BlinkMacSystemFont', 'Segoe UI', 'Roboto', 'sans-serif'],
        mono: ['Geist Mono', 'ui-monospace', 'monospace']
      },
      colors: {
        surface: {
          DEFAULT: '#09090b',
          subtle: '#121215',
          card: '#18181b',
          border: '#27272a',
          hover: '#222226'
        }
      }
    },
  },
  plugins: [],
}
