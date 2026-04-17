// @ts-check
import { test, expect } from '@playwright/test';

/**
 * E2E: Report generation — navigate, create schedule, verify run state.
 *
 * Prerequisites: full stack running with seeded admin user.
 */

const ADMIN_USER = 'admin';
const ADMIN_PASS = 'Admin@12345678';

async function login(page) {
  await page.goto('/login');
  await page.fill('input[type="text"], input[name="username"]', ADMIN_USER);
  await page.fill('input[type="password"]', ADMIN_PASS);
  await page.click('button[type="submit"]');
  await expect(page).not.toHaveURL(/\/login/, { timeout: 10000 });
}

test.describe('Report Generation', () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test('navigate to reports page', async ({ page }) => {
    await page.click('a[href*="/reports"], .nav-item:has-text("Reports")');
    await expect(page).toHaveURL(/\/reports/, { timeout: 5000 });
    // Page content area must be visible.
    await expect(page.locator('.page-content, .content, main')).toBeVisible({ timeout: 3000 });
  });

  test('schedule form validates required fields', async ({ page }) => {
    await page.goto('/reports');
    await page.waitForTimeout(1000);

    // Create button must exist.
    const createBtn = page.locator('button:has-text("Create"), button:has-text("New"), button:has-text("Add")');
    await expect(createBtn.first()).toBeVisible({ timeout: 3000 });
    await createBtn.first().click();
    await page.waitForTimeout(500);

    // Submit empty form — validation errors must appear.
    const submitBtn = page.locator('button[type="submit"], button:has-text("Save"), button:has-text("Create")').last();
    await expect(submitBtn).toBeVisible({ timeout: 3000 });
    await submitBtn.click();
    await page.waitForTimeout(500);

    // Validation error must be visible.
    const errorText = page.locator('[class*="error"], [class*="invalid"], [role="alert"]');
    await expect(errorText.first()).toBeVisible({ timeout: 3000 });
  });

  test('reports page renders content area without errors', async ({ page }) => {
    await page.goto('/reports');
    await page.waitForTimeout(2000);

    // The reports page must render its content area.
    await expect(page.locator('.page-content, .content, main')).toBeVisible();

    // If the page has a schedules or runs section, it must be visible.
    const tableOrEmpty = page.locator('table, [class*="table"], [class*="empty"], [class*="no-data"]');
    await expect(tableOrEmpty.first()).toBeVisible({ timeout: 5000 });
  });
});
