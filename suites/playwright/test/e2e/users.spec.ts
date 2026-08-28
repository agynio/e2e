import { argosScreenshot } from '@argos-ci/playwright';
import { test, expect } from './fixtures';
import { getMe } from './console-api';

test.describe('users', { tag: ['@svc_console'] }, () => {
  test('lists users', async ({ adminPage: page }) => {
    const me = await getMe(page);
    const userLabel = me.user?.name || me.user?.meta?.id;
    if (!userLabel) {
      throw new Error('GetMe response missing user label for users list.');
    }

    await page.goto('/users');
    // The directory arrives a page at a time and the box filters what has
    // arrived, so on an installation holding hundreds of users the one being
    // looked for has to be fetched before it can be found.
    await page.getByTestId('list-search').fill(userLabel);
    const userRow = page.getByTestId('users-row').filter({ hasText: userLabel });
    const loadMore = page.getByTestId('load-more');
    while ((await userRow.count()) === 0 && (await loadMore.count()) > 0) {
      await loadMore.click();
      await expect(loadMore).toBeEnabled({ timeout: 15000 });
    }
    await expect(userRow.first()).toBeVisible({ timeout: 15000 });
    await argosScreenshot(page, 'users-list');
  });

  test('shows user detail', async ({ adminPage: page }) => {
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
