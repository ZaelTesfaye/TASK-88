// @ts-check
import { test, expect } from '@playwright/test';

/**
 * E2E: Master data CRUD — create, view, deactivate a SKU record.
 *
 * Prerequisites: full stack running (docker-compose up) with seeded admin user.
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

  test('navigate to master data page and see table', async ({ page }) => {
    await page.click('a[href*="/master"], .nav-item:has-text("Master Data")');
    await expect(page).toHaveURL(/\/master/, { timeout: 5000 });
    // Table must be visible — not conditional.
    await expect(page.locator('table, [class*="table"], [class*="data-grid"]')).toBeVisible({ timeout: 5000 });
  });

  test('create a new SKU record', async ({ page }) => {
    await page.goto('/master/sku');
    await expect(page.locator('table, [class*="table"]')).toBeVisible({ timeout: 5000 });

    // Click create/add button — must exist.
    const addBtn = page.locator('button:has-text("Add"), button:has-text("Create"), button:has-text("New")');
    await expect(addBtn.first()).toBeVisible({ timeout: 3000 });
    await addBtn.first().click();

    // Fill in the form — code/key field must exist.
    const codeInput = page.locator('input[name="natural_key"], input[name="code"], input[placeholder*="code" i], input[placeholder*="key" i]');
    await expect(codeInput.first()).toBeVisible({ timeout: 3000 });

    const testCode = 'E2ETEST' + Date.now().toString().slice(-6);
    await codeInput.first().fill(testCode);

    // Submit the form and intercept the API response.
    const submitBtn = page.locator('button[type="submit"], button:has-text("Save"), button:has-text("Create")');
    const [createResponse] = await Promise.all([
      page.waitForResponse(r => r.url().includes('/api/v1/master/') && r.request().method() === 'POST'),
      submitBtn.last().click(),
    ]);
    expect(createResponse.status()).toBe(201);
    const createBody = await createResponse.json();
    expect(createBody).toHaveProperty('data');

    // Reload page to confirm persisted state.
    await page.reload();
    await page.waitForTimeout(500);
    const searchInput = page.locator('input[placeholder*="search" i], input[name="search"]');
    await expect(searchInput.first()).toBeVisible({ timeout: 3000 });
    await searchInput.first().fill(testCode);
    await page.waitForTimeout(500);
    await expect(page.locator(`text=${testCode}`)).toBeVisible({ timeout: 5000 });
  });

  test('deactivate a record shows reason dialog', async ({ page }) => {
    await page.goto('/master/sku');
    await expect(page.locator('table, [class*="table"]')).toBeVisible({ timeout: 5000 });

    // Deactivate button must exist on at least one row.
    const deactivateBtn = page.locator('button:has-text("Deactivate"), button[title*="deactivate" i]');
    await expect(deactivateBtn.first()).toBeVisible({ timeout: 5000 });
    await deactivateBtn.first().click();

    // Dialog must appear asking for a reason.
    await expect(
      page.locator('[class*="dialog"], [class*="modal"], [role="dialog"]')
    ).toBeVisible({ timeout: 3000 });

    // Reason textarea/input must be present.
    await expect(
      page.locator('textarea, input[name="reason"], input[placeholder*="reason" i]')
    ).toBeVisible();
  });
});
