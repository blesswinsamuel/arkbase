import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import tailwindcss from '@tailwindcss/vite';
import path from 'path';

export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      '@': path.resolve(import.meta.dirname, './src'),
    },
  },
  server: {
    host: '0.0.0.0',
    port: 18081,
    proxy: {
      '/api': 'http://localhost:18082',
      '/docs': 'http://localhost:18082',
      '/openapi.json': 'http://localhost:18082',
      '/metrics': 'http://localhost:18082',
    },
  },
});
