import { argosScreenshot } from '@argos-ci/playwright';
import { expect, test } from './fixtures';
import { createAgent, createOrganization, setSelectedOrganization } from './console-api';

const buildName = (prefix: string, now: number) => `${prefix}-${now}`;

test.describe('agents', { tag: ['@svc_console'] }, () => {
  test('shows agent configuration and edit dialog', async ({ page }) => {
    const now = Date.now();
    const organizationId = await createOrganization(page, buildName('e2e-org-agent', now));
    await setSelectedOrganization(page, organizationId);

    const agentName = buildName('e2e-agent', now);
    const agentNickname = `agent-${now}`;
    const agentId = await createAgent(page, {
      organizationId,
      name: agentName,
      role: 'assistant',
      description: 'E2E agent for visual snapshots',
      configuration: JSON.stringify({ greeting: 'hello' }),
      image: 'ghcr.io/agyn/agent:latest',
      initImage: 'ghcr.io/agyn/agent-init:latest',
    });

    await page.goto(`/organizations/${organizationId}/agents/${agentId}`);
    const configurationCard = page.getByTestId('agent-configuration-card');
    await expect(configurationCard).toBeVisible({ timeout: 15000 });
    await expect(configurationCard.getByText('Nickname')).toBeVisible({ timeout: 15000 });
    await argosScreenshot(page, 'agent-detail-configuration');

    await page.getByTestId('agent-configuration-edit').click();
    const dialog = page.getByTestId('agent-configuration-dialog');
    await expect(dialog).toBeVisible({ timeout: 15000 });
    const nicknameInput = dialog.getByTestId('agent-configuration-nickname');
    await expect(nicknameInput).toBeVisible({ timeout: 15000 });
    await nicknameInput.fill(agentNickname);
    await expect(nicknameInput).toHaveValue(agentNickname);
    await argosScreenshot(page, 'agent-edit-dialog', { fullPage: false });
  });

  test('shows agent create form', async ({ page }) => {
    const organizationId = await createOrganization(page, `e2e-org-agent-create-${Date.now()}`);
    await setSelectedOrganization(page, organizationId);

    await page.goto(`/organizations/${organizationId}/agents/new`);
    await expect(page.getByTestId('agent-create-form')).toBeVisible({ timeout: 15000 });
    await expect(page.getByTestId('agent-create-nickname')).toBeVisible({ timeout: 15000 });
    await argosScreenshot(page, 'agent-create-form');
  });
});
