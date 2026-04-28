import { test, expect } from '@playwright/test';

test.describe('Login flow', () => {
  test.beforeEach(async ({ page }) => {
    // Clear any existing auth state
    await page.goto('/');
    await page.evaluate(() => {
      localStorage.removeItem('token');
      localStorage.removeItem('user');
    });
    await page.reload();
  });

  test('login page renders with username and password fields', async ({
    page,
  }) => {
    // Should see the login form
    await expect(page.locator('text=AlphaPulse')).toBeVisible();
    await expect(page.locator('input[type="text"]')).toBeVisible();
    await expect(page.locator('input[type="password"]')).toBeVisible();
    await expect(page.locator('button[type="submit"]')).toBeVisible();
    // Labels
    await expect(page.locator('text=用户名')).toBeVisible();
    await expect(page.locator('text=密码')).toBeVisible();
  });

  test('successful login shows dashboard', async ({ page }) => {
    // Fill in credentials
    await page.locator('input[type="text"]').fill('admin');
    await page.locator('input[type="password"]').fill('admin123');
    await page.locator('button[type="submit"]').click();

    // Should navigate to dashboard — sidebar should appear
    await expect(
      page.locator('[role="navigation"][aria-label="主导航"]'),
    ).toBeVisible({ timeout: 15_000 });
    // Dashboard content should load
    await expect(page.locator('text=总览')).toBeVisible();
  });

  test('invalid credentials keep user on login page', async ({ page }) => {
    await page.locator('input[type="text"]').fill('baduser');
    await page.locator('input[type="password"]').fill('badpassword');
    await page.locator('button[type="submit"]').click();

    // Wait for the request to complete and any page reload to settle
    await page.waitForTimeout(3_000);

    // User should still see the login form (not navigated to dashboard)
    // The 401 interceptor may reload the page, but we should still be on login
    await expect(page.locator('input[type="password"]')).toBeVisible();
    // Sidebar should NOT appear (not authenticated)
    await expect(
      page.locator('[role="navigation"][aria-label="主导航"]'),
    ).not.toBeVisible({ timeout: 2_000 });
  });
});
