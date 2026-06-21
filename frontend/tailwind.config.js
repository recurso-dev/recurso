/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  darkMode: "class",
  theme: {
    extend: {
      colors: {
        "primary": "#000000", // Stark black for primary actions (Linear style)
        "primary-hover": "#333333", 
        "background-light": "#FAFAFA", // gray-50
        "background-dark": "#09090B", // zinc-950
        "surface-light": "#FFFFFF",
        "surface-dark": "#18181B", // zinc-900
        "border-light": "#E4E4E7", // zinc-200
        "border-dark": "#27272A", // zinc-800
        // Keeping legacy names for compatibility but updated values
        "text-light-primary": "#18181B", // zinc-900
        "text-light-secondary": "#71717A", // zinc-500
      },
      fontFamily: {
        "sans": ["Inter", "sans-serif"],
        "display": ["Inter", "sans-serif"]
      },
    },
  },
  plugins: [],
}
