import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import path from 'path'

export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  server: {
    host: '0.0.0.0',
    port: 5173,
    strictPort: true,
    hmr: {
      host: '100.100.116.63',
      port: 5173,
    },
    allowedHosts: true,
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
        timeout: 600000,
        proxyTimeout: 600000,
      },
      '/health': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
      // 把 /api/auth 也代理到后端（后端真实路径是 /login）
      '/api/auth': {
        target: 'http://localhost:8080',
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/api\/auth/, ''),
      },
    },
  },
  build: {
    rollupOptions: {
      output: {
        manualChunks(id: string) {
          if (id.includes('node_modules/')) {
            // Core React runtime — loaded on every page
            if (id.includes('/react/') || id.includes('/react-dom/')) {
              return 'vendor';
            }
            // ECharts and wrapper — used by multiple chart-heavy pages
            if (id.includes('/echarts/') || id.includes('/echarts-for-react/')) {
              return 'charts';
            }
            // Lightweight charts — only used by KlinePage, separate from ECharts
            if (id.includes('/lightweight-charts/')) {
              return 'lightweight-charts';
            }
            // Animation library — used across many pages
            if (id.includes('/framer-motion/')) {
              return 'motion';
            }
            // Icon library — used by Layout and most pages
            if (id.includes('/lucide-react/')) {
              return 'icons';
            }
          }
        },
      },
    },
  },
})
