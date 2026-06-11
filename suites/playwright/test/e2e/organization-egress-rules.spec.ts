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
    const secretId = await createSecret(page, {
      providerId,
      name: `e2e-egress-secret-${now}`,
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

    await page.goto(`/organizations/${organizationId}/agents/${agentId}`);
    await expect(page.getByTestId('agent-egress-rule-attachments-heading')).toBeVisible({ timeout: 15000 });
    await expect(page.getByTestId('agent-egress-rule-attachment-row').filter({ hasText: ruleName })).toBeVisible({ timeout: 15000 });
  });
});
