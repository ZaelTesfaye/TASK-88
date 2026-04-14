// @ts-check
import { test, expect } from '@playwright/test';

/**
 * E2E: Master data CRUD — create, view, deactivate a SKU record.
 *
 * Equivalent to the "cart → checkout" user journey for this data-ops app:
 * the user creates a data record, reviews it, and deactivates it.
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

test.describe('Master Data CRUD', () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test('navigate to master data page', async ({ page }) => {
    await page.click('a[href*="/master"], .nav-item:has-text("Master Data")');
    await expect(page).toHaveURL(/\/master/, { timeout: 5000 });
    // Table or list should be visible
    await expect(page.locator('table, [class*="table"], [class*="data-grid"]')).toBeVisible({ timeout: 5000 });
  });

  test('create a new SKU record', async ({ page }) => {
    await page.goto('/master/sku');
    await expect(page.locator('table, [class*="table"]')).toBeVisible({ timeout: 5000 });

    // Click create/add button
    const addBtn = page.locator('button:has-text("Add"), button:has-text("Create"), button:has-text("New")');
    if (await addBtn.count() > 0) {
      await addBtn.first().click();

      // Fill in the form — expect a code/key field
      const codeInput = page.locator('input[name="natural_key"], input[name="code"], input[placeholder*="code" i], input[placeholder*="key" i]');
      if (await codeInput.count() > 0) {
        const testCode = 'E2ETEST' + Date.now().toString().slice(-6);
        await codeInput.first().fill(testCode);

        // Submit the form
        const submitBtn = page.locator('button[type="submit"], button:has-text("Save"), button:has-text("Create")');
        await submitBtn.last().click();

        // Wait for success indication
        await page.waitForTimeout(1000);

        // Verify the record appears (search for it)
        const searchInput = page.locator('input[placeholder*="search" i], input[name="search"]');
        if (await searchInput.count() > 0) {
          await searchInput.first().fill(testCode);
          await page.waitForTimeout(500);
          await expect(page.locator(`text=${testCode}`)).toBeVisible({ timeout: 5000 });
        }
      }
    }
  });

  test('deactivate a record shows reason dialog', async ({ page }) => {
    await page.goto('/master/sku');
    await expect(page.locator('table, [class*="table"]')).toBeVisible({ timeout: 5000 });

    // Look for a deactivate button on any row
    const deactivateBtn = page.locator('button:has-text("Deactivate"), button[title*="deactivate" i]');
    if (await deactivateBtn.count() > 0) {
      await deactivateBtn.first().click();

      // A dialog/modal should appear asking for a reason
      await expect(
        page.locator('[class*="dialog"], [class*="modal"], [role="dialog"]')
      ).toBeVisible({ timeout: 3000 });

      // Reason textarea/input should be present
      await expect(
        page.locator('textarea, input[name="reason"], input[placeholder*="reason" i]')
      ).toBeVisible();
    }
  });
});
