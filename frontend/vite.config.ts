import react from '@vitejs/plugin-react';
import type { InlineConfig as VitestInlineConfig } from 'vitest/node';
import { defineConfig } from 'vite';

const testConfig: {
  test: VitestInlineConfig;
} = {
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: './src/test/setup.ts',
  },
};

export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
  },
  ...testConfig,
});