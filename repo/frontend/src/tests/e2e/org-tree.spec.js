// @ts-check
import { test, expect } from '@playwright/test';

/**
 * E2E: Org tree management — create node, verify hierarchy, delete node.
 *
 * Prerequisites: full stack running with seeded admin + root org node.
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

test.describe('Org Tree Management', () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test('navigate to org tree page and see root node', async ({ page }) => {
    await page.click('a[href*="/org"], .nav-item:has-text("Org Tree")');
    await expect(page).toHaveURL(/\/org/, { timeout: 5000 });

    // Tree must render — root org node must be visible.
    await expect(
      page.locator('text=Root Organisation, text=root, [class*="tree"]')
    ).toBeVisible({ timeout: 5000 });
  });

  test('create node form validates required fields', async ({ page }) => {
    await page.goto('/org');
    await page.waitForTimeout(1000);

    // Add button must exist.
    const addBtn = page.locator('button:has-text("Add"), button:has-text("Create"), button:has-text("New")');
    await expect(addBtn.first()).toBeVisible({ timeout: 3000 });
    await addBtn.first().click();
    await page.waitForTimeout(500);

    // Submit empty — validation errors must appear.
    const submitBtn = page.locator(
      'button[type="submit"], button:has-text("Save"), button:has-text("Create")'
    ).last();
    await expect(submitBtn).toBeVisible({ timeout: 3000 });
    await submitBtn.click();
    await page.waitForTimeout(500);

    const errors = page.locator('[class*="error"], [class*="invalid"]');
    await expect(errors.first()).toBeVisible({ timeout: 3000 });
  });

  test('create a child node under root', async ({ page }) => {
    await page.goto('/org');
    await page.waitForTimeout(1000);

    // Add button must exist.
    const addBtn = page.locator('button:has-text("Add"), button:has-text("Create"), button:has-text("New")');
    await expect(addBtn.first()).toBeVisible({ timeout: 3000 });
    await addBtn.first().click();
    await page.waitForTimeout(500);

    // Name input must exist.
    const nameInput = page.locator('input[name="name"], input[placeholder*="name" i]');
    await expect(nameInput.first()).toBeVisible({ timeout: 3000 });

    const nodeName = 'E2E Region ' + Date.now().toString().slice(-4);
    await nameInput.first().fill(nodeName);

    // Fill level_code if present.
    const levelSelect = page.locator('select[name="level_code"], [class*="select"]:has-text("Level")');
    if (await levelSelect.count() > 0) {
      await levelSelect.first().selectOption({ label: 'Region' }).catch(() => {});
    }

    // Fill level_label if present.
    const labelInput = page.locator('input[name="level_label"]');
    if (await labelInput.count() > 0) {
      await labelInput.first().fill('Region');
    }

    const submitBtn = page.locator(
      'button[type="submit"], button:has-text("Save"), button:has-text("Create")'
    ).last();
    await submitBtn.click();
    await page.waitForTimeout(1500);

    // Verify the new node appears in the tree — hard assertion.
    await expect(page.locator(`text=${nodeName}`)).toBeVisible({ timeout: 5000 });
  });

  test('delete node shows confirmation warning', async ({ page }) => {
    await page.goto('/org');
    await page.waitForTimeout(1000);

    // Delete button must exist on at least one node.
    const deleteBtn = page.locator(
      'button:has-text("Delete"), button[title*="delete" i], button[aria-label*="delete" i]'
    );
    await expect(deleteBtn.first()).toBeVisible({ timeout: 5000 });
    await deleteBtn.first().click();

    // Confirmation dialog must appear.
    await expect(
      page.locator('[class*="dialog"], [class*="modal"], [role="dialog"]')
    ).toBeVisible({ timeout: 3000 });
  });
});
