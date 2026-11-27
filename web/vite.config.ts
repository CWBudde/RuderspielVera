import path from "node:path";
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";
import { VitePWA } from "vite-plugin-pwa";

// Absolute URL of the deployed site; Open Graph and Twitter cards need absolute image URLs.
const siteUrl = (process.env.VITE_SITE_URL || "https://cwbudde.github.io/RuderspielVera/").replace(/\/?$/, "/");

export default defineConfig(() => ({
  // VITE_BASE: public path for static hosting, e.g. "/RuderspielVera/" on GitHub Pages.
  base: process.env.VITE_BASE || "/",
  plugins: [
    react(),
    tailwindcss(),
    {
      name: "site-url",
      transformIndexHtml: (html) => html.replaceAll("%SITE_URL%", siteUrl),
    },
    VitePWA({
      // public/manifest.webmanifest is linked from index.html directly.
      manifest: false,
      // A new version waits until every tab is closed: swapping chunks under a running game would
      // break lazy imports, and saved games live in localStorage anyway.
      registerType: "prompt",
      injectRegister: "script-defer",
      workbox: {
        // Precache the app shell, the art and the wasm engine so a started game runs offline.
        globPatterns: ["**/*.{js,css,html,svg,png,jpg,webp,woff2,wasm,webmanifest}"],
        globIgnores: ["404.html", "og-image.jpg"],
        maximumFileSizeToCacheInBytes: 12 * 1024 * 1024,
        // wasmTransport appends ?v=<hash> for HTTP cache busting; the precache has its own revisions.
        ignoreURLParametersMatching: [/^v$/],
        navigateFallback: "index.html",
        navigateFallbackDenylist: [/\/api\//],
        cleanupOutdatedCaches: true,
      },
    }),
  ],
  resolve: {
    alias: [{ find: "@", replacement: path.resolve(import.meta.dirname, "./src") }],
  },
  server: {
    proxy: {
      "/api": { target: "http://localhost:8080", changeOrigin: true },
    },
  },
}));
