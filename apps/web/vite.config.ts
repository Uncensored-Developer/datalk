import react from "@vitejs/plugin-react";
import { defineConfig } from "vite";

const backendProxyTarget =
  process.env.VITE_API_PROXY_TARGET ?? "http://localhost:8007";
const backendStaticDist = "../backend/servers/echo/staticweb/dist";

export default defineConfig({
  plugins: [react()],
  cacheDir: ".vite",
  build: {
    outDir: backendStaticDist,
    emptyOutDir: true,
  },
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
