import { defineConfig, mergeConfig } from 'vitest/config';
import viteConfig from './vite.config.js';

export default mergeConfig(
  viteConfig,
  defineConfig({
    test: {
      environment: 'jsdom',
      globals: true,
      setupFiles: ['./src/test/setup.js'],
      include: ['src/**/*.{test,spec}.{js,ts}'],
      // Exclude Playwright E2E specs — they use @playwright/test, not vitest,
      // and must be run via `npx playwright test`.
      exclude: ['**/node_modules/**', 'src/tests/e2e/**'],
      coverage: {
        provider: 'v8',
        reporter: ['text', 'json', 'json-summary', 'html'],
        include: ['src/**/*.{js,vue}'],
        exclude: ['src/test/**', 'src/main.js'],
      },
    },
  })
);
