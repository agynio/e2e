import { argosScreenshot } from '@argos-ci/playwright';
import { test, expect } from './fixtures';
import { createOrganization, setSelectedOrganization } from './console-api';

test.describe('organizations', { tag: ['@svc_console'] }, () => {
  test('lists organizations', async ({ page }) => {
    const orgName = `e2e-org-list-${Date.now()}`;
    await createOrganization(page, orgName);

    await page.goto('/organizations');
    await expect(page.getByTestId('organizations-table')).toBeVisible({ timeout: 15000 });
    await expect(page.getByTestId('organizations-row').filter({ hasText: orgName })).toBeVisible({ timeout: 15000 });
    await argosScreenshot(page, 'organizations-list');
  });

  test('org detail shows overview', async ({ page }) => {
    const orgName = `e2e-org-detail-${Date.now()}`;
    const orgId = await createOrganization(page, orgName);
    await setSelectedOrganization(page, orgId);

    await page.goto(`/organizations/${orgId}`);
    await expect(page.getByTestId('page-title')).toHaveText('Overview', { timeout: 15000 });
    await expect(page.getByTestId('organization-overview-card')).toHaveCount(7);
    await argosScreenshot(page, 'organization-detail-overview');
  });
});
