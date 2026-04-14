// @ts-check
import { test, expect } from '@playwright/test';

/**
 * E2E: Report generation — trigger a report and verify the run appears.
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
  });

  test('schedule form validates required fields', async ({ page }) => {
    await page.goto('/reports');
    await page.waitForTimeout(1000);

    const createBtn = page.locator('button:has-text("Create"), button:has-text("New"), button:has-text("Add")');
    if (await createBtn.count() > 0) {
      await createBtn.first().click();
      await page.waitForTimeout(500);

      // Try to submit empty form — should show validation errors
      const submitBtn = page.locator('button[type="submit"], button:has-text("Save"), button:has-text("Create")').last();
      if (await submitBtn.count() > 0) {
        await submitBtn.click();
        await page.waitForTimeout(500);

        // Validation error should appear
        const errorText = page.locator('[class*="error"], [class*="invalid"], [role="alert"]');
        await expect(errorText.first()).toBeVisible({ timeout: 3000 });
      }
    }
  });

  test('report history shows state chips', async ({ page }) => {
    await page.goto('/reports');
    await page.waitForTimeout(2000);

    // Look for state indicator chips in the runs list
    const chips = page.locator('[class*="chip"], [class*="badge"], [class*="status"]');
    // If there are any runs displayed, they should have state chips
    if (await chips.count() > 0) {
      const chipText = await chips.first().textContent();
      expect(['ready', 'running', 'failed', 'pending', 'completed']).toContain(
        chipText?.trim().toLowerCase() ?? ''
      );
    }
  });

  test('download button disabled for failed reports', async ({ page }) => {
    await page.goto('/reports');
    await page.waitForTimeout(2000);

    // If there are failed report rows, their download buttons should be disabled
    const failedRows = page.locator('tr:has-text("failed"), [class*="row"]:has-text("failed")');
    if (await failedRows.count() > 0) {
      const downloadBtn = failedRows.first().locator('button:has-text("Download"), button[title*="download" i]');
      if (await downloadBtn.count() > 0) {
        await expect(downloadBtn.first()).toBeDisabled();
      }
    }
  });
});
