import { argosScreenshot } from '@argos-ci/playwright';
import { test, expect } from './fixtures';

test.describe('dashboard', { tag: ['@svc_console', '@smoke'] }, () => {
  test('dashboard shows stats cards', async ({ page }) => {
    const usersLoaded = page.waitForResponse(
      (resp) => resp.url().includes('ListUsers') && resp.status() === 200,
      { timeout: 15000 },
    );
    const orgsLoaded = page.waitForResponse(
      (resp) => resp.url().includes('ListOrganizations') && resp.status() === 200,
      { timeout: 15000 },
    );
    const runnersLoaded = page.waitForResponse(
      (resp) => resp.url().includes('ListRunners') && resp.status() === 200,
      { timeout: 15000 },
    );

    await page.goto('/');
    await Promise.all([usersLoaded, orgsLoaded, runnersLoaded]);

    await expect(page.getByTestId('page-title')).toHaveText('Dashboard', { timeout: 15000 });
    await expect(page.getByTestId('dashboard-stat-card')).toHaveCount(3);
    await argosScreenshot(page, 'dashboard-overview');
  });
});
