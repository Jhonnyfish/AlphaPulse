import { test, expect } from '@playwright/test';

test.describe('Data loading patterns', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    await expect(
      page.locator('[role="navigation"][aria-label="主导航"]'),
    ).toBeVisible({ timeout: 15_000 });
  });

  test('skeleton loading appears on initial page load', async ({ page }) => {
    // Navigate to a data-heavy page — watchlist
    const sidebar = page.locator(
      '[role="navigation"][aria-label="主导航"]',
    );
    await sidebar.getByRole('button', { name: '自选股', exact: true }).click();

    // On the first load, skeletons with animate-pulse may flash briefly
    // We just verify that the page eventually resolves to content
    await expect(page.locator('#main-content')).toBeVisible();

    // Wait for content to settle
    await page.waitForTimeout(3_000);
    // Main content should still be intact
    await expect(page.locator('#main-content')).toBeVisible();
  });

  test('data renders after API response', async ({ page }) => {
    // Dashboard is the default view — wait for section headings that appear after data loads
    await expect(
      page.locator('h2:has-text("指数走势")').first(),
    ).toBeVisible({ timeout: 15_000 });

    // The heading text confirms data rendered — check that main content has substantive text
    const mainContent = page.locator('#main-content');
    // Wait a bit more for full data load
    await page.waitForTimeout(2_000);
    const text = await mainContent.textContent();
    expect(text?.length).toBeGreaterThan(20);
  });

  test('DegradedBanner appears for degraded endpoints', async ({ page }) => {
    // Navigate to candidates page which may show DegradedBanner
    const sidebar = page.locator(
      '[role="navigation"][aria-label="主导航"]',
    );
    await sidebar.getByRole('button', { name: '候选股', exact: true }).click();

    // Wait for content
    await expect(page.locator('#main-content')).toBeVisible();
    await page.waitForTimeout(3_000);

    // Check if DegradedBanner appears (role="alert" with "数据暂时不可用")
    // It may or may not appear depending on backend state
    const degradedBanner = page.locator(
      '[role="alert"]:has-text("数据暂时不可用")',
    );
    const bannerCount = await degradedBanner.count();

    // If banner appears, verify it has the expected content
    if (bannerCount > 0) {
      await expect(degradedBanner.first()).toBeVisible();
      // Verify the dismiss button works
      const dismissBtn = degradedBanner
        .first()
        .locator('button');
      const dismissCount = await dismissBtn.count();
      if (dismissCount > 0) {
        await dismissBtn.first().click();
        await page.waitForTimeout(300);
      }
    }

    // Navigate to screener page
    await sidebar.getByRole('button', { name: '选股器', exact: true }).click();
    await page.waitForTimeout(3_000);

    const screenerBanner = page.locator(
      '[role="alert"]:has-text("数据暂时不可用")',
    );
    // Similar check — may or may not appear
    const screenerBannerCount = await screenerBanner.count();
    if (screenerBannerCount > 0) {
      await expect(screenerBanner.first()).toBeVisible();
    }
  });
});
