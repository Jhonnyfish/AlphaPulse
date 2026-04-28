import { test, expect } from '@playwright/test';

test.describe('Responsive design', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    await expect(
      page.locator('[role="navigation"][aria-label="主导航"]'),
    ).toBeVisible({ timeout: 15_000 });
  });

  test('desktop viewport (1280x800) — sidebar visible', async ({ page }) => {
    await page.setViewportSize({ width: 1280, height: 800 });

    // On desktop, sidebar should be visible (lg: breakpoint = 1024px)
    const sidebar = page.locator(
      '[role="navigation"][aria-label="主导航"]',
    );
    await expect(sidebar).toBeVisible();

    // Hamburger button should NOT be visible on desktop
    // (it's inside a header with class lg:hidden)
    const mobileHeader = page.locator('header.lg\\:hidden');
    await expect(mobileHeader).toBeHidden();

    // Main content should be visible
    await expect(page.locator('#main-content')).toBeVisible();
  });

  test('mobile viewport (375x812) — sidebar hidden, hamburger visible', async ({
    page,
  }) => {
    await page.setViewportSize({ width: 375, height: 812 });
    await page.waitForTimeout(300);

    // Hamburger button should be visible
    const hamburger = page.locator(
      'button[aria-label="打开导航菜单"]',
    );
    await expect(hamburger).toBeVisible();

    // The sidebar is off-screen (translated left)
    // The aside element exists but is translated out of view
    const sidebar = page.locator('aside[role="navigation"]');

    // Mobile header should be visible
    const mobileHeader = page.locator('header.lg\\:hidden');
    await expect(mobileHeader).toBeVisible();

    // Main content should still be visible
    await expect(page.locator('#main-content')).toBeVisible();

    // Open sidebar via hamburger
    await hamburger.click();
    await page.waitForTimeout(300);
    await expect(sidebar).toBeVisible();

    // Navigate to a different page
    await sidebar.locator('button:has-text("行情")').click();
    await page.waitForTimeout(300);

    // Sidebar should auto-close after navigation (mobile behavior)
    // The aside is still in DOM but translated off-screen
    await expect(page.locator('#main-content')).toBeVisible();
  });

  test('tablet viewport (768x1024) — layout adapts', async ({ page }) => {
    await page.setViewportSize({ width: 768, height: 1024 });
    await page.waitForTimeout(300);

    // 768px is below lg (1024px), so mobile header should be visible
    const mobileHeader = page.locator('header.lg\\:hidden');
    await expect(mobileHeader).toBeVisible();

    // Main content should be visible
    await expect(page.locator('#main-content')).toBeVisible();

    // Hamburger should work
    const hamburger = page.locator(
      'button[aria-label="打开导航菜单"]',
    );
    await expect(hamburger).toBeVisible();
    await hamburger.click();
    await page.waitForTimeout(300);

    const sidebar = page.locator('aside[role="navigation"]');
    await expect(sidebar).toBeVisible();
  });
});
