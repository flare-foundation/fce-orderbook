import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      "/direct": {
        target: "http://localhost:6665",
        changeOrigin: true,
      },
      "/state": {
        target: "http://localhost:6665",
        changeOrigin: true,
      },
      "/action": {
        target: "http://localhost:6665",
        changeOrigin: true,
      },
    },
  },
});
