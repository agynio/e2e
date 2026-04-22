import { argosScreenshot } from '@argos-ci/playwright';
import { test, expect } from './fixtures';

test.describe('settings', { tag: ['@svc_console'] }, () => {
  test('shows settings profile info', async ({ page }) => {
    await page.goto('/');
    await expect(page.getByTestId('user-menu-trigger')).toBeVisible({ timeout: 15000 });
    await page.getByTestId('user-menu-trigger').click();
    await page.getByTestId('user-menu-settings').click();
    await expect(page).toHaveURL(/\/settings$/);
    await expect(page.getByTestId('settings-profile-card')).toBeVisible({ timeout: 15000 });
    await argosScreenshot(page, 'settings-profile');
  });
});
