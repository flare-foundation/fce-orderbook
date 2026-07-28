import { defineConfig, loadEnv } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), "VITE_");
  const upstream = env.VITE_PROXY_UPSTREAM || "http://localhost:6674";
  // xaman-service (FSA/XRPL wallet backend) — see ../xaman-service/.
  const xamanUpstream = env.VITE_XAMAN_UPSTREAM || "http://localhost:8787";
  return {
    plugins: [react()],
    server: {
      proxy: {
        "/direct": { target: upstream, changeOrigin: true },
        "/state": { target: upstream, changeOrigin: true },
        "/action": { target: upstream, changeOrigin: true },
        "/login": { target: xamanUpstream, changeOrigin: true },
        "/sign": { target: xamanUpstream, changeOrigin: true },
        "/payload": { target: xamanUpstream, changeOrigin: true },
        "/relay": { target: xamanUpstream, changeOrigin: true },
      },
    },
  };
});
