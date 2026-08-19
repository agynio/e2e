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

    // The organization_id the Terraform provider needs, without reading the URL.
    // The same wait as the assertions above. These three sit in one block that
    // renders together, and giving the last of them the 5s default made it the
    // only one that could lose a race the others had already won.
    await expect(page.getByTestId('organization-overview-identity')).toContainText(orgName, { timeout: 15000 });
    await expect(page.getByTestId('organization-overview-id')).toHaveText(orgId, { timeout: 15000 });
    await expect(page.getByTestId('organization-overview-id-copy')).toBeVisible({ timeout: 15000 });

    await argosScreenshot(page, 'organization-detail-overview');
  });
});
