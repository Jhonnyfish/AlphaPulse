import { test, expect } from '@playwright/test';

/**
 * Smoke test for all major pages.
 * For each page: click sidebar item → wait for content → verify no crash.
 */

interface PageSmoke {
  label: string;
  /** Text content expected to appear after the page loads */
  expectedContent?: string;
}

const PAGES: PageSmoke[] = [
  { label: '总览', expectedContent: '指数走势' },
  { label: '自选股' },
  { label: '行情' },
  { label: '个股分析' },
  { label: '热门概念' },
  { label: '板块' },
  { label: '信号' },
  { label: '候选股' },
  { label: '选股器' },
  { label: '持仓' },
  { label: '交易日志' },
  { label: '策略回测' },
  { label: '资金流向' },
  { label: 'K线' },
  { label: '对比' },
];

test.describe('Page smoke tests', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    await expect(
      page.locator('[role="navigation"][aria-label="主导航"]'),
    ).toBeVisible({ timeout: 15_000 });
  });

  for (const pg of PAGES) {
    test(`page "${pg.label}" loads without crash`, async ({ page }) => {
      const sidebar = page.locator(
        '[role="navigation"][aria-label="主导航"]',
      );
      const navButton = sidebar.locator(`button:has-text("${pg.label}")`);
      await expect(navButton).toBeVisible({ timeout: 5_000 });
      await navButton.click();

      // Wait for main content to be present
      const main = page.locator('#main-content');
      await expect(main).toBeVisible({ timeout: 10_000 });

      // Verify no error boundary crash (the error boundary shows a message
      // like "出错了" or "Something went wrong")
      const errorBoundary = page.locator(
        'text=/出错了|Something went wrong|页面加载失败/i',
      );
      await page.waitForTimeout(1_000);
      await expect(errorBoundary).toHaveCount(0);

      // If expected content is specified, verify it appears
      if (pg.expectedContent) {
        await expect(
          page.locator(`text=${pg.expectedContent}`).first(),
        ).toBeVisible({ timeout: 15_000 });
      }

      // Verify no uncaught JS errors in console
      const errors: string[] = [];
      page.on('pageerror', (err) => errors.push(err.message));
      await page.waitForTimeout(500);
      // We allow some time for errors to accumulate; filter out known benign ones
      const seriousErrors = errors.filter(
        (e) =>
          !e.includes('ResizeObserver') &&
          !e.includes('NetworkError') &&
          !e.includes('Failed to fetch'),
      );
      expect(seriousErrors).toHaveLength(0);
    });
  }
});
