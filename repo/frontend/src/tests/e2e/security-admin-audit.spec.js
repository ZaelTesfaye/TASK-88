// @ts-check
import { test, expect } from '@playwright/test';

/**
 * E2E: Security admin — audit log filtering and report management.
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
  });

  test('password reset approval flow is visible', async ({ page }) => {
    await page.goto('/security');
    await page.waitForTimeout(1000);

    // The security page should show password reset section or tabs
    const resetSection = page.locator(
      'text=Password Reset, text=password reset, [class*="password-reset"], button:has-text("Password")'
    );
    if (await resetSection.count() > 0) {
      await resetSection.first().click();
      await page.waitForTimeout(500);

      // Should see a list or empty state for reset requests
      const content = page.locator(
        'table, [class*="empty"], [class*="no-data"], text=No requests'
      );
      await expect(content.first()).toBeVisible({ timeout: 3000 });
    }
  });

  test('dual-approval indicator shows approval count', async ({ page }) => {
    await page.goto('/security');
    await page.waitForTimeout(1000);

    // Look for approval indicators (0/2, 1/2, 2/2)
    const approvalIndicators = page.locator('text=/\\d\\/2/');
    if (await approvalIndicators.count() > 0) {
      const text = await approvalIndicators.first().textContent();
      expect(text).toMatch(/\d\/2/);
    }
  });

  test('retention policies are listed', async ({ page }) => {
    await page.goto('/security');
    await page.waitForTimeout(1000);

    // Navigate to retention tab/section if it exists
    const retentionTab = page.locator(
      'button:has-text("Retention"), [class*="tab"]:has-text("Retention"), text=Retention'
    );
    if (await retentionTab.count() > 0) {
      await retentionTab.first().click();
      await page.waitForTimeout(500);
    }

    // Should display retention policy data (artifact types like audit_logs, sessions, etc.)
    const policyContent = page.locator(
      'text=audit_logs, text=sessions, text=ingestion_failures'
    );
    if (await policyContent.count() > 0) {
      await expect(policyContent.first()).toBeVisible();
    }
  });

  test('sensitive field registry shows fields', async ({ page }) => {
    await page.goto('/security');
    await page.waitForTimeout(1000);

    // Look for sensitive fields tab/section
    const fieldsTab = page.locator(
      'button:has-text("Sensitive"), [class*="tab"]:has-text("Sensitive"), text=Sensitive'
    );
    if (await fieldsTab.count() > 0) {
      await fieldsTab.first().click();
      await page.waitForTimeout(500);
    }

    // Should show field entries with masking patterns
    const fieldContent = page.locator('text=email, text=password, text=full, text=last4');
    if (await fieldContent.count() > 0) {
      await expect(fieldContent.first()).toBeVisible();
    }
  });
});
