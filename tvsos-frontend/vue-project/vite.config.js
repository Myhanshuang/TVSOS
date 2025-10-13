// vite.config.js
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import path from 'path'  // <-- 引入 path，用于别名解析

export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src')  // <-- 配置 @ 指向 src
    }
  },
  server: {
    proxy: {
      // 只要请求路径以 /api 开头，就会被代理到 http://localhost:8080
      '/api': {
        target: 'http://127.0.0.1:4523/m1/7124866-6847792-6235619',
        changeOrigin: true,
        // 可以加上 rewrite，如果后端不需要 /api 前缀：
        rewrite: (path) => path.replace(/^\/api/, '')
      }
    }
  }
})