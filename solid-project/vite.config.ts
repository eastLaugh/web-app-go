import { defineConfig } from 'vite';
import solidPlugin from 'vite-plugin-solid';
import devtools from 'solid-devtools/vite';

export default defineConfig({
  plugins: [devtools(), solidPlugin()],
  base: '/app/',
  server: {
    port: 3000,
  },
  build: {
    target: 'esnext',
    outDir: '../go/dist',
    emptyOutDir: true,
  },
});
