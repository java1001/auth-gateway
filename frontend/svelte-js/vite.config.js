import { defineConfig, loadEnv } from 'vite'
import { svelte } from '@sveltejs/vite-plugin-svelte'
import tailwindcss from '@tailwindcss/vite'

export default defineConfig(({ mode }) => {
  // Load env file based on `mode` in the current working directory.
  // Set the third parameter to '' to load all env regardless of the `VITE_` prefix.
  const env = loadEnv(mode, process.cwd(), '')
  
  return {
    plugins: [svelte(), tailwindcss()],
    server: {
      port: parseInt(env.PORT) || 5173,
    },
    define: {
      // Expose these explicitly to the client bundle
      'import.meta.env.APP_URL': JSON.stringify(env.APP_URL || 'http://localhost:5173'),
      'import.meta.env.API_URL': JSON.stringify(env.API_URL || 'http://localhost:8080/auth'),
    }
  }
})
