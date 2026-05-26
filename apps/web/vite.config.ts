import react from "@vitejs/plugin-react";
import { defineConfig } from "vite";

const backendProxyTarget =
  process.env.VITE_API_PROXY_TARGET ?? "http://localhost:8007";
const buildOutDir = process.env.VITE_BUILD_OUT_DIR ?? "dist";

export default defineConfig({
  plugins: [react()],
  cacheDir: ".vite",
  build: {
    outDir: buildOutDir,
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
