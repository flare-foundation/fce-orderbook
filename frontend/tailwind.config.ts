import type { Config } from "tailwindcss";

export default {
  content: ["./index.html", "./src/**/*.{ts,tsx}"],
  theme: {
    extend: {
      colors: {
        bid: "#16a34a",
        ask: "#dc2626",
      },
    },
  },
  plugins: [],
} satisfies Config;
