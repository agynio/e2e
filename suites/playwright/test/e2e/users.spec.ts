import { argosScreenshot } from '@argos-ci/playwright';
import { test, expect } from './fixtures';
import { getMe } from './console-api';

test.describe('users', { tag: ['@svc_console'] }, () => {
  test('lists users', async ({ page }) => {
    const me = await getMe(page);
    const userLabel = me.user?.name || me.user?.meta?.id;
    if (!userLabel) {
      throw new Error('GetMe response missing user label for users list.');
    }

    await page.goto('/users');
    const userRow = page.getByTestId('users-row').filter({ hasText: userLabel });
    await expect(userRow).toBeVisible({ timeout: 15000 });
    await argosScreenshot(page, 'users-list');
  });

  test('shows user detail', async ({ page }) => {
    const me = await getMe(page);
    const identityId = me.user?.meta?.id;
    if (!identityId) {
      throw new Error('GetMe response missing identity id for user detail.');
    }

    await page.goto(`/users/${identityId}`);
    await expect(page.getByTestId('user-profile-card')).toBeVisible({ timeout: 15000 });
    await argosScreenshot(page, 'user-detail');
  });
});
