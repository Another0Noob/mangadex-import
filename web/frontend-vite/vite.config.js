import { defineConfig } from "vite";

export default defineConfig({
  server: {
    port: 39390,
    cors: {
      // Allow Go backend to access Vite dev server
      origin: "http://localhost:39039",
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
