/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ["./web/src/**/*.{html,js,svelte}"],
  safelist: [
    "cais-password-wrap",
    "cais-password-toggle",
    "cais-chat-scroll-down",
    "cais-thinking",
    "cais-thinking-dots",
    "cais-select-search",
    "cais-select-search-native",
    "cais-select-search-trigger",
    "cais-select-search-panel",
    "cais-select-search-input",
    "cais-select-search-list",
    "cais-select-search-option",
    "cais-select-search-label",
    "cais-select-search-chevron",
    "is-selected",
    "is-highlighted",
    "is-hidden",
  ],
  theme: {
    extend: {
      fontFamily: {
        sans: ["ui-sans-serif", "system-ui", "-apple-system", "Segoe UI", "sans-serif"],
        display: ["ui-sans-serif", "system-ui", "-apple-system", "Segoe UI", "sans-serif"],
        mono: ['"JetBrains Mono"', "ui-monospace", "SFMono-Regular", "monospace"],
      },
      boxShadow: {
        "2xs": "0 1px 2px 0 rgb(0 0 0 / 0.05)",
        xs: "0 1px 2px 0 rgb(0 0 0 / 0.05)",
      },
    },
  },
  plugins: [],
};
