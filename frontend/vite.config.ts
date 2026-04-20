import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      "/direct": {
        target: process.env.VITE_PROXY_UPSTREAM || "http://localhost:6674",
        changeOrigin: true,
      },
      "/state": {
        target: process.env.VITE_PROXY_UPSTREAM || "http://localhost:6674",
        changeOrigin: true,
      },
      "/action": {
        target: process.env.VITE_PROXY_UPSTREAM || "http://localhost:6674",
        changeOrigin: true,
      },
    },
  },
});
