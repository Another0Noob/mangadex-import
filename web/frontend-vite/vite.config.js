import { defineConfig } from "vite";

export default defineConfig({
  server: {
    port: 39390,
    proxy: {
      "/api": "http://localhost:39039",
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
