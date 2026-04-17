// @ts-check
import { test, expect } from '@playwright/test';

/**
 * E2E: Login flow
 *
 * Prerequisites: full stack running (docker-compose up).
 * Default admin credentials: admin / Admin@12345678
 */

const ADMIN_USER = 'admin';
const ADMIN_PASS = 'Admin@12345678';

test.describe('Login', () => {
  test('shows login page when unauthenticated', async ({ page }) => {
    await page.goto('/');
    // Should redirect to /login
    await expect(page).toHaveURL(/\/login/);
    await expect(page.locator('input[type="text"], input[name="username"]')).toBeVisible();
    await expect(page.locator('input[type="password"]')).toBeVisible();
  });

  test('rejects invalid credentials', async ({ page }) => {
    await page.goto('/login');
    await page.fill('input[type="text"], input[name="username"]', 'wronguser');
    await page.fill('input[type="password"]', 'WrongPassword123!');
    await page.click('button[type="submit"]');

    // Should show an error message and stay on login
    await expect(page.locator('[class*="error"], [role="alert"]')).toBeVisible({ timeout: 5000 });
    await expect(page).toHaveURL(/\/login/);
  });

  test('successful login redirects to app and returns token', async ({ page }) => {
    await page.goto('/login');
    await page.fill('input[type="text"], input[name="username"]', ADMIN_USER);
    await page.fill('input[type="password"]', ADMIN_PASS);

    // Intercept the login API call to verify backend response.
    const [response] = await Promise.all([
      page.waitForResponse(r => r.url().includes('/api/v1/auth/login') && r.request().method() === 'POST'),
      page.click('button[type="submit"]'),
    ]);
    expect(response.status()).toBe(200);
    const body = await response.json();
    expect(body).toHaveProperty('token');
    expect(body).toHaveProperty('user');
    expect(body.user).toHaveProperty('username', ADMIN_USER);

    // Should navigate away from /login to the main app.
    await expect(page).not.toHaveURL(/\/login/, { timeout: 10000 });

    // Sidebar should be visible — confirms persisted auth state.
    await expect(page.locator('.sidebar, nav, [class*="sidebar"]')).toBeVisible();
  });

  test('logout returns to login page', async ({ page }) => {
    // Login first
    await page.goto('/login');
    await page.fill('input[type="text"], input[name="username"]', ADMIN_USER);
    await page.fill('input[type="password"]', ADMIN_PASS);
    await page.click('button[type="submit"]');
    await expect(page).not.toHaveURL(/\/login/, { timeout: 10000 });

    // Click user menu trigger then logout
    await page.click('.user-menu-trigger, .user-menu button, [class*="user-avatar"]');
    await page.click('button:has-text("Logout"), [class*="logout"]');

    await expect(page).toHaveURL(/\/login/, { timeout: 5000 });
  });
});
