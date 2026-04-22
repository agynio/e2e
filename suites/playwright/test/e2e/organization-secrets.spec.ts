import { argosScreenshot } from '@argos-ci/playwright';
import { test, expect } from './fixtures';
import {
  clearOrganizationSecrets,
  createOrganization,
  createSecret,
  createSecretProvider,
  setSelectedOrganization,
} from './console-api';

test.describe('organization-secrets', { tag: ['@svc_console'] }, () => {
  test('shows empty secrets tab initially', async ({ page }) => {
    const organizationId = await createOrganization(page, `e2e-org-secrets-${Date.now()}`);
    await setSelectedOrganization(page, organizationId);
    await clearOrganizationSecrets(page, organizationId);

    await page.goto(`/organizations/${organizationId}/secret-providers`);
    await expect(page.getByTestId('secret-providers-empty')).toBeVisible({ timeout: 15000 });
    await page.goto(`/organizations/${organizationId}/secrets`);
    await expect(page.getByTestId('secrets-empty')).toBeVisible({ timeout: 15000 });
  });

  test('shows secret providers and secrets', async ({ page }) => {
    const organizationId = await createOrganization(page, `e2e-org-secrets-list-${Date.now()}`);
    await setSelectedOrganization(page, organizationId);
    const providerName = `e2e-provider-${Date.now()}`;
    const secretName = `e2e-secret-${Date.now()}`;

    const providerId = await createSecretProvider(page, {
      organizationId,
      name: providerName,
      url: 'https://vault.example.com',
    });
    await createSecret(page, {
      providerId,
      name: secretName,
      value: `e2e-value-${Date.now()}`,
      organizationId,
    });

    await page.goto(`/organizations/${organizationId}/secret-providers`);
    await expect(page.getByTestId('secret-provider-row').filter({ hasText: providerName })).toBeVisible({ timeout: 15000 });
    await page.goto(`/organizations/${organizationId}/secrets`);
    await expect(page.getByTestId('secret-row').filter({ hasText: secretName })).toBeVisible({ timeout: 15000 });
    await argosScreenshot(page, 'organization-secrets-list');
  });
});
