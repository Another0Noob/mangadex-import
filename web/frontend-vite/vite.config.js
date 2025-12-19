import { defineConfig } from "vite";

export default defineConfig({
  server: {
    port: 5173,
    strictPort: true,
    cors: {
      // Allow Go backend to access Vite dev server
      origin: "http://localhost:3939",
    },
  },

  build: {
    manifest: true,
    // Disable the modulepreload polyfill
    modulePreload: {
      polyfill: false,
    },
  },
});
