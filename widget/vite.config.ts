import path from "node:path"
import { fileURLToPath } from "node:url"
import { defineConfig } from "vite"
import react from "@vitejs/plugin-react"
import tailwindcss from "@tailwindcss/vite"

const rootDir = path.dirname(fileURLToPath(import.meta.url))

export default defineConfig({
  plugins: [react(), tailwindcss()],
  // Don't copy public/ into the lib build; postbuild handles ncore.png
  publicDir: false,
  define: {
    "process.env.NODE_ENV": JSON.stringify("production"),
  },
  build: {
    outDir: path.resolve(rootDir, "../internal/static/widget"),
    emptyOutDir: true,
    lib: {
      entry: path.resolve(rootDir, "src/main.tsx"),
      name: "NcoreWidget",
      formats: ["es"],
      fileName: () => "widget.js",
    },
    rollupOptions: {
      output: {
        assetFileNames: "widget.[ext]",
      },
    },
    cssCodeSplit: false,
  },
})
