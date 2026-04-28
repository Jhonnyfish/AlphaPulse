import { test as setup, expect } from '@playwright/test';
import { fileURLToPath } from 'url';
import path from 'path';
import fs from 'fs';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const authDir = path.join(__dirname, '.auth');
const authFile = path.join(authDir, 'user.json');

setup('authenticate via API', async ({ page }) => {
  // Ensure .auth directory exists
  if (!fs.existsSync(authDir)) {
    fs.mkdirSync(authDir, { recursive: true });
  }

  // Login via API to get a token
  const apiResponse = await page.request.post(
    'http://localhost:8899/api/auth/login',
    {
      data: {
        username: 'admin',
        password: 'admin123',
      },
    },
  );
  expect(apiResponse.ok()).toBeTruthy();
  const body = await apiResponse.json();
  const token: string = body.token;

  // Navigate to the app and inject the token into localStorage
  await page.goto('/');
  await page.evaluate((t) => {
    localStorage.setItem('token', t);
    localStorage.setItem(
      'user',
      JSON.stringify({ id: '1', username: 'admin', role: 'admin' }),
    );
  }, token);

  // Reload so the app picks up the token
  await page.reload();
  // Wait for the app to show authenticated content (sidebar navigation)
  await expect(
    page.locator('[role="navigation"][aria-label="主导航"]'),
  ).toBeVisible({ timeout: 15_000 });

  // Save storage state for reuse by other tests
  await page.context().storageState({ path: authFile });
});
