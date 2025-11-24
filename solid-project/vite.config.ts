import { defineConfig } from 'vite';
import solidPlugin from 'vite-plugin-solid';
import devtools from 'solid-devtools/vite';
import postsPlugin from './vite-plugin-posts';

export default defineConfig({
  plugins: [devtools(), solidPlugin(), postsPlugin()],
  base: '/app/',
  server: {
    port: 3000,
  },
  build: {
    target: 'esnext',
    outDir: '../go/cmd/server/dist',
    emptyOutDir: true,
  },
});
