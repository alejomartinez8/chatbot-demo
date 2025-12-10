import path from "path"
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

export default defineConfig({
  /* Plugins */
  plugins: [
    react(),
    tailwindcss(),
  ],
  /* Paths */
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
})
