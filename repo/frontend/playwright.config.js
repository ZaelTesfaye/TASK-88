import { defineConfig, devices } from '@playwright/test';

/**
 * Playwright E2E test configuration for Multi-Org Hub.
 *
 * Expects the full stack to be running:
 *   - Frontend on http://localhost:3000
 *   - Backend  on http://localhost:8080
 *   - MySQL    on localhost:3306
 *
 * Start with:  docker-compose up -d
 * Run tests:   npx playwright test
 */
export default defineConfig({
  testDir: './src/tests/e2e',
  fullyParallel: false,          // serial — tests share auth state
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 1 : 0,
  workers: 1,
  reporter: [['list'], ['html', { open: 'never' }]],

  use: {
    baseURL: process.env.E2E_BASE_URL || 'http://localhost:3000',
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
    video: 'retain-on-failure',
  },

  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],
});
