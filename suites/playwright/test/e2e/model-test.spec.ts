import { argosScreenshot } from '@argos-ci/playwright';
import { expect, test } from './fixtures';
import { createLLMProvider, createModel, createOrganization, setSelectedOrganization } from './console-api';

const TEST_LLM_ENDPOINT = 'https://testllm.dev/v1/org/agynio/suite/agn/responses';

test.describe('model-test', { tag: ['@svc_console'] }, () => {
  test('tests model successfully', async ({ page }) => {
    const organizationId = await createOrganization(page, `e2e-org-model-test-success-${Date.now()}`);
    await setSelectedOrganization(page, organizationId);

    const providerId = await createLLMProvider(page, {
      organizationId,
      endpoint: TEST_LLM_ENDPOINT,
      authMethod: 'AUTH_METHOD_X_API_KEY',
      token: 'e2e-test-token',
      protocol: 'PROTOCOL_RESPONSES',
    });

    const modelName = `e2e-model-success-${Date.now()}`;
    await createModel(page, {
      organizationId,
      providerId,
      name: modelName,
      remoteName: 'summarize-history',
    });

    await page.goto(`/organizations/${organizationId}/models`);
    const row = page.getByTestId('organization-model-row').filter({ hasText: modelName });
    await expect(row).toBeVisible({ timeout: 15000 });

    await row.getByTestId('organization-model-test').click();
    await expect(page.getByTestId('organization-model-test-pending')).toBeVisible();
    await expect(page.getByTestId('organization-model-test-success')).toBeVisible({ timeout: 15000 });
    await expect(page.getByTestId('organization-model-test-output')).toHaveText(/\S+/);
    await argosScreenshot(page, 'model-test-success');
  });

  test('shows model test failure', async ({ page }) => {
    const organizationId = await createOrganization(page, `e2e-org-model-test-failure-${Date.now()}`);
    await setSelectedOrganization(page, organizationId);

    const providerId = await createLLMProvider(page, {
      organizationId,
      endpoint: TEST_LLM_ENDPOINT,
      authMethod: 'AUTH_METHOD_X_API_KEY',
      token: 'e2e-test-token',
      protocol: 'PROTOCOL_RESPONSES',
    });

    const modelName = `e2e-model-failure-${Date.now()}`;
    await createModel(page, {
      organizationId,
      providerId,
      name: modelName,
      remoteName: 'nonexistent-model',
    });

    await page.goto(`/organizations/${organizationId}/models`);
    const row = page.getByTestId('organization-model-row').filter({ hasText: modelName });
    await expect(row).toBeVisible({ timeout: 15000 });

    await row.getByTestId('organization-model-test').click();
    await expect(page.getByTestId('organization-model-test-failure')).toBeVisible({ timeout: 15000 });
    await expect(page.getByTestId('organization-model-test-error')).toHaveText(/\S+/);
    await argosScreenshot(page, 'model-test-failure');
  });
});
