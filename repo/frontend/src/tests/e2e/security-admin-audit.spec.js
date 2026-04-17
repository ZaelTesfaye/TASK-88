// @ts-check
import { test, expect } from '@playwright/test';

/**
 * E2E: Security admin — audit log, retention policies, sensitive fields.
 *
 * Prerequisites: full stack running with seeded admin user and default
 * retention policies / sensitive field entries from init.sql.
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

test.describe('Security Admin & Audit', () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test('navigate to security admin page', async ({ page }) => {
    await page.click('a[href*="/security"], .nav-item:has-text("Security")');
    await expect(page).toHaveURL(/\/security/, { timeout: 5000 });
    // Page content must render.
    await expect(page.locator('.page-content, .content, main')).toBeVisible({ timeout: 3000 });
  });

  test('password reset section is accessible', async ({ page }) => {
    await page.goto('/security');
    await page.waitForTimeout(1000);

    // The security page must show password reset section or tab.
    const resetSection = page.locator(
      'text=Password Reset, text=password reset, [class*="password-reset"], button:has-text("Password")'
    );
    await expect(resetSection.first()).toBeVisible({ timeout: 5000 });
    await resetSection.first().click();
    await page.waitForTimeout(500);

    // Must show a list or empty state for reset requests — hard assertion.
    const content = page.locator(
      'table, [class*="empty"], [class*="no-data"], text=No requests'
    );
    await expect(content.first()).toBeVisible({ timeout: 3000 });
  });

  test('retention policies are listed', async ({ page }) => {
    await page.goto('/security');
    await page.waitForTimeout(1000);

    // Navigate to retention tab/section.
    const retentionTab = page.locator(
      'button:has-text("Retention"), [class*="tab"]:has-text("Retention"), text=Retention'
    );
    await expect(retentionTab.first()).toBeVisible({ timeout: 5000 });
    await retentionTab.first().click();
    await page.waitForTimeout(500);

    // Seeded retention policies must be visible (from init.sql: audit_logs, sessions, etc.).
    const policyContent = page.locator(
      'text=audit_logs, text=sessions, text=ingestion_failures'
    );
    await expect(policyContent.first()).toBeVisible({ timeout: 5000 });
  });

  test('sensitive field registry shows seeded fields', async ({ page }) => {
    await page.goto('/security');
    await page.waitForTimeout(1000);

    // Navigate to sensitive fields tab/section.
    const fieldsTab = page.locator(
      'button:has-text("Sensitive"), [class*="tab"]:has-text("Sensitive"), text=Sensitive'
    );
    await expect(fieldsTab.first()).toBeVisible({ timeout: 5000 });
    await fieldsTab.first().click();
    await page.waitForTimeout(500);

    // Seeded sensitive fields must be visible (from init.sql: email, password_hash, etc.).
    const fieldContent = page.locator('text=email, text=password, text=full, text=last4');
    await expect(fieldContent.first()).toBeVisible({ timeout: 5000 });
  });
});
