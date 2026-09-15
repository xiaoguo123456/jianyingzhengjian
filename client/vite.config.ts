import { defineConfig } from 'vite'
import uni from '@dcloudio/vite-plugin-uni'

// The uni CLI does not forward --port, so the H5 dev port comes from VITE_DEV_PORT.
// Default 5173; the live-backend preview uses 5175 (see .claude/launch.json).
export default defineConfig({
  plugins: [uni()],
  server: {
    host: '127.0.0.1',
    port: Number(process.env.VITE_DEV_PORT) || 5173,
    strictPort: true,
  },
})
