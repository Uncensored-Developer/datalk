import react from "@vitejs/plugin-react";
import { defineConfig } from "vite";

const backendProxyTarget =
  process.env.VITE_API_PROXY_TARGET ?? "http://localhost:8007";

export default defineConfig({
  plugins: [react()],
  cacheDir: ".vite",
  server: {
    port: 5173,
    proxy: {
      "/api": {
        target: backendProxyTarget,
        changeOrigin: true,
      },
    },
  },
});
