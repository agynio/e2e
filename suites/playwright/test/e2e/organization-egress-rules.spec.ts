import { test, expect } from './fixtures';
import {
  createAgent,
  createEgressRule,
  createEgressRuleAttachment,
  createOrganization,
  createSecret,
  createSecretProvider,
  setSelectedOrganization,
} from './console-api';

function suffix(): string {
  return `${Date.now()}`;
}

test.describe('organization-egress-rules', { tag: ['@svc_console'] }, () => {
  test('manages egress rules and agent attachments', async ({ page }) => {
    const now = suffix();
    const organizationId = await createOrganization(page, `e2e-org-egress-${now}`);
    await setSelectedOrganization(page, organizationId);

    const providerId = await createSecretProvider(page, {
      organizationId,
      name: `e2e-egress-provider-${now}`,
      url: 'https://vault.example.com',
    });
    const secretName = `e2e-egress-secret-${now}`;
    const secretId = await createSecret(page, {
      providerId,
      name: secretName,
      value: `egress/token/${now}`,
      organizationId,
    });
    const ruleName = `e2e-egress-rule-${now}`;
    const ruleId = await createEgressRule(page, {
      organizationId,
      name: ruleName,
      domainPattern: `api-${now}.example.com`,
      secretId,
    });
    const agentId = await createAgent(page, {
      organizationId,
      name: `e2e-egress-agent-${now}`,
      role: 'assistant',
      image: 'ghcr.io/agyn/agent:latest',
      initImage: 'ghcr.io/agyn/agent-init:latest',
    });
    await createEgressRuleAttachment(page, { ruleId, agentId });

    await page.goto(`/organizations/${organizationId}/egress-rules`);
    await expect(page.getByTestId('egress-rules-heading')).toBeVisible({ timeout: 15000 });
    await expect(page.getByTestId('egress-rule-row').filter({ hasText: ruleName })).toBeVisible({ timeout: 15000 });
    await expect(page.getByTestId('nav-organization-egress-rules')).toBeVisible();

    const uiRuleName = `e2e-egress-ui-rule-${now}`;
    await page.getByTestId('egress-rules-create-button').click();
    await expect(page.getByTestId('egress-rules-create-dialog')).toBeVisible({ timeout: 15000 });
    await page.getByTestId('egress-rules-create-name').fill(uiRuleName);
    await page.getByTestId('egress-rules-create-domain').fill(`ui-${now}.example.com`);
    await page.getByTestId('egress-rules-create-ports').fill('443');
    await page.getByTestId('egress-rules-create-methods').fill('GET');
    await page.getByTestId('egress-rules-create-path').fill('/repos/**');
    await page.getByTestId('egress-rules-create-add-header').click();
    await page.getByTestId('egress-rules-create-header-name').fill('Authorization');
    await page.getByTestId('egress-rules-create-header-scheme').click();
    await page.getByRole('option', { name: 'Bearer' }).click();
    await page.getByTestId('egress-rules-create-header-source').click();
    await page.getByRole('option', { name: 'Secret' }).click();
    await page.getByTestId('egress-rules-create-header-secret-search').fill(secretName);
    await page.getByTestId('egress-rules-create-header-secret').click();
    await page.getByRole('option', { name: secretName }).click();
    await page.getByTestId('egress-rules-create-submit').click();
    await expect(page.getByTestId('egress-rule-row').filter({ hasText: uiRuleName })).toBeVisible({ timeout: 15000 });

    await page.getByTestId('egress-rules-create-button').click();
    await expect(page.getByTestId('egress-rules-create-dialog')).toBeVisible({ timeout: 15000 });
    await page.getByTestId('egress-rules-create-name').fill(`e2e-egress-invalid-${now}`);
    await page.getByTestId('egress-rules-create-submit').click();
    await expect(page.getByText('Domain pattern is required.')).toBeVisible({ timeout: 15000 });
    await page.getByTestId('egress-rules-create-add-header').click();
    await page.getByTestId('egress-rules-create-header-name').fill('Authorization');
    await page.getByTestId('egress-rules-create-domain').fill(`invalid-${now}.example.com`);
    await page.getByTestId('egress-rules-create-submit').click();
    await expect(page.getByText('Each header requires a name and literal value or selected secret.')).toBeVisible({ timeout: 15000 });
    await page.getByTestId('egress-rules-create-cancel').click();

    await page.goto(`/organizations/${organizationId}/agents/${agentId}`);
    await expect(page.getByTestId('agent-egress-rule-attachments-heading')).toBeVisible({ timeout: 15000 });
    await expect(page.getByTestId('agent-egress-rule-attachment-row').filter({ hasText: ruleName })).toBeVisible({ timeout: 15000 });
  });
});
