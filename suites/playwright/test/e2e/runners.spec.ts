import { argosScreenshot } from '@argos-ci/playwright';
import { test, expect } from './fixtures';
import { createOrganization, listRunners, registerRunner, setSelectedOrganization } from './console-api';

test.describe('runners', { tag: ['@svc_console'] }, () => {
  test('lists cluster runners', async ({ page }) => {
    await page.goto('/runners');
    await expect(page.getByTestId('runners-table')).toBeVisible({ timeout: 15000 });
    await expect(page.getByTestId('runners-row').first()).toBeVisible({ timeout: 15000 });
    await argosScreenshot(page, 'runners-list');
  });

  test('lists organization and cluster runners', async ({ page }) => {
    const orgId = await createOrganization(page, `e2e-org-runners-${Date.now()}`);
    await setSelectedOrganization(page, orgId);
    const orgRunnerName = `e2e-org-runner-${Date.now()}`;
    const clusterRunnerName = `e2e-cluster-runner-${Date.now()}`;

    await registerRunner(page, {
      name: orgRunnerName,
      organizationId: orgId,
      labels: { scope: 'organization' },
    });
    await registerRunner(page, {
      name: clusterRunnerName,
      labels: { scope: 'cluster' },
    });

    await page.goto(`/organizations/${orgId}/runners`);
    await expect(page.getByTestId('organization-runners-table')).toBeVisible({ timeout: 15000 });
    await expect(
      page.getByTestId('organization-runner-row').filter({ hasText: orgRunnerName }),
    ).toBeVisible({ timeout: 15000 });
    await expect(page.getByTestId('organization-cluster-runners-table')).toBeVisible({ timeout: 15000 });
    await expect(
      page.getByTestId('organization-cluster-runner-row').filter({ hasText: clusterRunnerName }),
    ).toBeVisible({ timeout: 15000 });
    await argosScreenshot(page, 'organization-runners-list');
  });

  test('organization runner detail shows metadata', async ({ page }) => {
    const orgId = await createOrganization(page, `e2e-org-runner-detail-${Date.now()}`);
    await setSelectedOrganization(page, orgId);
    const orgRunnerName = `e2e-org-runner-detail-${Date.now()}`;
    const runner = await registerRunner(page, {
      name: orgRunnerName,
      organizationId: orgId,
      labels: { scope: 'organization' },
    });
    const runnerId = runner.meta?.id;
    if (!runnerId) {
      throw new Error('RegisterRunner response missing runner id for org detail test.');
    }

    await page.goto(`/organizations/${orgId}/runners/${runnerId}`);
    await expect(page.getByTestId('runner-details-card')).toBeVisible({ timeout: 15000 });
    await argosScreenshot(page, 'organization-runner-detail');
  });

  test('runner detail shows metadata', async ({ page }) => {
    const runners = await listRunners(page);
    const runnerId = runners[0]?.meta?.id;
    if (!runnerId) {
      test.skip(true, 'No runners available for detail view.');
      return;
    }

    await page.goto(`/runners/${runnerId}`);
    await expect(page.getByTestId('runner-details-card')).toBeVisible({ timeout: 15000 });
    await argosScreenshot(page, 'runner-detail');
  });
});
