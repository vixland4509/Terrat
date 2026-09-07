/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        tokyo: {
          bg: "#1a1b26",
          header: "#13141c",
          surface: "#16161e",
          border: "#292e42",
          subtle: "#24283b",
          fg: "#c0caf5",
          muted: "#565f89",
          accent: "#7aa2f7",
          green: "#9ece6a",
          yellow: "#e0af68",
          red: "#f7768e",
          cyan: "#7dcfff",
          magenta: "#bb9af7",
        }
      },
      fontFamily: {
        mono: ['"JetBrains Mono"', '"Fira Code"', 'ui-monospace', 'SFMono-Regular', 'Menlo', 'Monaco', 'Consolas', 'monospace'],
        sans: ['"Space Grotesk"', 'system-ui', '-apple-system', 'sans-serif'],
      }
    },
  },
  plugins: [],
}
