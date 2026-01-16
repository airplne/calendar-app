/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        // Protection-first color palette (from UX spec)
        'focus-block': '#0D9488', // Teal for protected zones
        'event': '#64748B',       // Slate gray for standard events
        'interruption': '#F59E0B', // Amber for attention without alarm
      },
    },
  },
  plugins: [],
}
