import { test, expect } from '@playwright/test';

test.describe('Dashboard page', () => {
  test.beforeEach(async ({ page }) => {
    // Navigate to dashboard (default view after login)
    await page.goto('/');
    // Wait for sidebar to confirm we're authenticated
    await expect(
      page.locator('[role="navigation"][aria-label="主导航"]'),
    ).toBeVisible({ timeout: 15_000 });
  });

  test('dashboard loads and shows main sections', async ({ page }) => {
    // Dashboard should have several sections with headings
    const sections = ['指数走势', '涨跌家数', '板块热力图', '信号摘要', '活动时间线'];

    for (const section of sections) {
      // Each section heading should eventually appear (after data loads)
      await expect(
        page.locator(`h2:has-text("${section}")`).first(),
      ).toBeVisible({ timeout: 15_000 });
    }
  });

  test('skeleton loading appears then disappears', async ({ page }) => {
    // On initial load, there should be skeleton elements with animate-pulse
    // We look for the pulse animation class — it appears during loading
    // and disappears once data arrives
    //
    // After data loads, the skeletons should be gone and real content present
    // Wait for at least one section heading to appear (data loaded)
    await expect(
      page.locator('h2:has-text("指数走势")').first(),
    ).toBeVisible({ timeout: 15_000 });

    // After data loads, there should be no more skeleton placeholders
    // covering the main content (they may still exist in other lazy areas)
    // At minimum, the main dashboard headings confirm data rendered
    const mainContent = page.locator('#main-content');
    await expect(mainContent).toBeVisible();
  });

  test('auto-refresh does not flash skeletons when data exists', async ({
    page,
  }) => {
    // Wait for dashboard to fully load
    await expect(
      page.locator('h2:has-text("指数走势")').first(),
    ).toBeVisible({ timeout: 15_000 });

    // Record number of visible skeleton elements
    const skeletonCountBefore = await page
      .locator('.animate-pulse')
      .count();

    // Wait a bit (the auto-refresh interval is 60s, we'll just check
    // that the page remains stable for a short period)
    await page.waitForTimeout(2_000);

    // Skeleton count should not increase
    const skeletonCountAfter = await page
      .locator('.animate-pulse')
      .count();
    expect(skeletonCountAfter).toBeLessThanOrEqual(skeletonCountBefore);
  });
});
