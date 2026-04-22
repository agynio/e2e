const { test, expect } = require('@playwright/test');

test.describe('Smoke', { tag: ['@smoke'] }, () => {
  test('renders placeholder content', async ({ page }) => {
    await page.setContent('<h1>Smoke</h1>');
    await expect(page.locator('h1')).toHaveText('Smoke');
  });
});
