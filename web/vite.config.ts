import path from "node:path";
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";

export default defineConfig(({ command }) => ({
  // VITE_BASE: public path for static hosting, e.g. "/RuderspielVera/" on GitHub Pages.
  base: process.env.VITE_BASE || "/",
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: [
      // Named portraits of real club members (src/assets/art/people/, git-ignored) are a dev-server
      // feature only: every build gets the empty stub, so the photos are never bundled or published.
      ...(command === "build"
        ? [
            {
              find: "@/components/peopleArt",
              replacement: path.resolve(import.meta.dirname, "./src/components/peopleArt.stub.ts"),
            },
          ]
        : []),
      { find: "@", replacement: path.resolve(import.meta.dirname, "./src") },
    ],
  },
  server: {
    proxy: {
      "/api": { target: "http://localhost:8080", changeOrigin: true },
    },
  },
}));
