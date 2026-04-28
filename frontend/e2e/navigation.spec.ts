import { test, expect } from '@playwright/test';

/** Sidebar items we expect to find — exact labels from Layout.tsx navItems */
const SIDEBAR_ITEMS = [
  '总览',
  '自选股',
  '行情',
  'K线',
  '个股分析',
  '板块',
  '对比',
  '资金流向',
  '趋势',
  '候选股',
  '选股器',
  '热门概念',
  '持仓',
  '交易日志',
  '信号',
  '自选分析',
  '资讯',
  '设置',
];

test.describe('Sidebar navigation', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    // Wait for authenticated content to load
    await expect(
      page.locator('[role="navigation"][aria-label="主导航"]'),
    ).toBeVisible({ timeout: 15_000 });
  });

  test('all main sidebar items are clickable and load views', async ({
    page,
  }) => {
    const sidebar = page.locator(
      '[role="navigation"][aria-label="主导航"]',
    );
    await expect(sidebar).toBeVisible();

    for (const label of SIDEBAR_ITEMS) {
      // Use exact role matching to avoid substring conflicts (e.g. "趋势" vs "多周期趋势")
      const navButton = sidebar.getByRole('button', { name: label, exact: true });
      await expect(navButton).toBeVisible({ timeout: 5_000 });
      await navButton.click();

      // After clicking, main content area should still exist (no crash)
      await expect(page.locator('#main-content')).toBeVisible({
        timeout: 10_000,
      });

      // Small delay for view transition
      await page.waitForTimeout(300);
    }
  });

  test('active nav item has aria-current="page"', async ({ page }) => {
    const sidebar = page.locator(
      '[role="navigation"][aria-label="主导航"]',
    );

    // Dashboard should be active by default
    const dashboardBtn = sidebar.getByRole('button', { name: '总览', exact: true });
    await expect(dashboardBtn).toHaveAttribute('aria-current', 'page');

    // Click watchlist
    const watchlistBtn = sidebar.getByRole('button', { name: '自选股', exact: true });
    await watchlistBtn.click();
    await page.waitForTimeout(300);

    // Watchlist should now be active
    await expect(watchlistBtn).toHaveAttribute('aria-current', 'page');
    // Dashboard should no longer be active
    await expect(dashboardBtn).not.toHaveAttribute('aria-current', 'page');
  });

  test('mobile hamburger menu opens sidebar', async ({ page }) => {
    // Set mobile viewport
    await page.setViewportSize({ width: 375, height: 812 });
    // Reload to pick up new viewport
    await page.goto('/');
    await page.waitForTimeout(1000);

    // Hamburger button should be visible on mobile
    const hamburger = page.locator(
      'button[aria-label="打开导航菜单"]',
    );
    await expect(hamburger).toBeVisible({ timeout: 10_000 });

    // Click hamburger to open sidebar
    await hamburger.click();
    await page.waitForTimeout(300);

    // Sidebar should now be visible
    const sidebar = page.locator(
      '[role="navigation"][aria-label="主导航"]',
    );
    await expect(sidebar).toBeVisible();

    // Click close button to close sidebar
    const closeBtn = page.locator('button[aria-label="关闭导航菜单"]');
    await expect(closeBtn).toBeVisible();
    await closeBtn.click();
    await page.waitForTimeout(300);
  });
});
