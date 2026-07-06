/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        brand: {
          DEFAULT: '#10B981',
          light: '#34D399',
          dark: '#059669',
        },
        surface: {
          DEFAULT: '#0c0e14',   // page background
          75: '#0f121a',        // alternate section background
          100: '#11141d',       // card background
          200: '#161a26',       // raised card / hover background
          300: '#1c2130',       // active background
        },
        line: {
          DEFAULT: '#1f2330',   // default border
          strong: '#2c3244',    // hover / emphasized border
        },
        fg: {
          DEFAULT: '#e8eaf0',   // headings
          muted: '#9ba1b0',     // body copy
          subtle: '#6b7280',    // captions / labels
        },
      },
      fontFamily: {
        sans: ['Inter', 'ui-sans-serif', 'system-ui', '-apple-system', 'sans-serif'],
        mono: ['"JetBrains Mono"', 'ui-monospace', 'SFMono-Regular', 'Menlo', 'monospace'],
      },
      maxWidth: {
        site: '80rem',
      },
    },
  },
  plugins: [],
}
