import { argosScreenshot } from '@argos-ci/playwright';
import type { Page } from '@playwright/test';
import { test, expect } from './fixtures';
import {
  createAgent,
  createHook,
  createImagePullSecret,
  createMcp,
  createOrganization,
  setSelectedOrganization,
} from './console-api';

const buildName = (prefix: string, now: number) => `${prefix}-${now}`;
const buildMcpName = (prefix: string, now: number) => buildName(prefix, now).replace(/-/g, '_');

async function prepareAgentFixture(page: Page, suffix: string) {
  const now = Date.now();
  const organizationId = await createOrganization(page, buildName(`e2e-org-${suffix}`, now));
  await setSelectedOrganization(page, organizationId);

  const agentName = buildName(`e2e-agent-${suffix}`, now);
  const agentId = await createAgent(page, {
    organizationId,
    name: agentName,
    role: 'assistant',
    image: 'ghcr.io/agyn/agent:latest',
    initImage: 'ghcr.io/agyn/agent-init:latest',
  });

  const registry = `ghcr.io/e2e/${suffix}`;
  const username = buildName('e2e-user', now);
  await createImagePullSecret(page, {
    organizationId,
    registry,
    username,
    value: buildName('e2e-token', now),
    description: `E2E image pull secret for ${suffix}`,
  });

  return {
    organizationId,
    agentId,
    registry,
    username,
    now,
  };
}

async function openImagePullSecretsDialog(opts: {
  page: Page;
  rowTestId: string;
  rowLabel: string;
  manageTestId: string;
  menuTestId: string;
  menuItemTestId: string;
}) {
  const row = opts.page.getByTestId(opts.rowTestId).filter({ hasText: opts.rowLabel });
  await expect(row).toBeVisible({ timeout: 15000 });
  await row.getByTestId(opts.manageTestId).click();
  await expect(opts.page.getByTestId(opts.menuTestId)).toBeVisible({ timeout: 15000 });
  await opts.page.getByTestId(opts.menuItemTestId).click();
  await expect(opts.page.getByTestId('nested-image-pull-secrets-dialog')).toBeVisible({ timeout: 15000 });
}

async function attachSecret(page: Page, secretLabel: string) {
  await expect(page.getByTestId('nested-image-pull-secrets-attach')).toBeEnabled({ timeout: 15000 });
  await page.getByTestId('nested-image-pull-secrets-select').click();
  await page.getByRole('option', { name: secretLabel }).click();
  await page.getByTestId('nested-image-pull-secrets-attach').click();
}

test.describe('agent-image-pull-secrets', { tag: ['@svc_console'] }, () => {
  test('manages MCP image pull secrets dialog', async ({ page }) => {
    const { organizationId, agentId, registry, username, now } = await prepareAgentFixture(page, 'mcp');
    const mcpName = buildMcpName('e2e-mcp', now);
    await createMcp(page, {
      agentId,
      name: mcpName,
      image: 'ghcr.io/agyn/mcp:latest',
      command: 'python -m mcp_server',
      description: 'E2E MCP',
    });

    await page.goto(`/organizations/${organizationId}/agents/${agentId}`);
    await openImagePullSecretsDialog({
      page,
      rowTestId: 'agent-mcp-row',
      rowLabel: mcpName,
      manageTestId: 'agent-mcp-manage',
      menuTestId: 'agent-mcp-manage-menu',
      menuItemTestId: 'agent-mcp-image-pull-secrets',
    });

    const dialog = page.getByTestId('nested-image-pull-secrets-dialog');
    await expect(dialog.getByText('No image pull secrets attached.')).toBeVisible({ timeout: 15000 });
    await argosScreenshot(page, 'agent-mcp-image-pull-secrets-empty', { fullPage: false });

    const secretLabel = `${registry} (${username})`;
    await attachSecret(page, secretLabel);

    const attachedRow = dialog.getByTestId('nested-image-pull-secret-row').filter({ hasText: registry });
    await expect(attachedRow).toBeVisible({ timeout: 15000 });
    await argosScreenshot(page, 'agent-mcp-image-pull-secrets-attached', { fullPage: false });

    await attachedRow.getByTestId('nested-image-pull-secret-detach').click();
    await expect(attachedRow).toHaveCount(0, { timeout: 15000 });
    await expect(dialog.getByText('No image pull secrets attached.')).toBeVisible({ timeout: 15000 });
    await argosScreenshot(page, 'agent-mcp-image-pull-secrets-detached', { fullPage: false });
  });

  test('manages Hook image pull secrets dialog', async ({ page }) => {
    const { organizationId, agentId, registry, username, now } = await prepareAgentFixture(page, 'hook');
    const hookEvent = buildName('e2e-hook-event', now);
    await createHook(page, {
      agentId,
      event: hookEvent,
      functionName: 'handleHook',
      image: 'ghcr.io/agyn/hook:latest',
      description: 'E2E hook',
    });

    await page.goto(`/organizations/${organizationId}/agents/${agentId}`);
    await openImagePullSecretsDialog({
      page,
      rowTestId: 'agent-hook-row',
      rowLabel: hookEvent,
      manageTestId: 'agent-hook-manage',
      menuTestId: 'agent-hook-manage-menu',
      menuItemTestId: 'agent-hook-image-pull-secrets',
    });

    const dialog = page.getByTestId('nested-image-pull-secrets-dialog');
    await expect(dialog.getByText('No image pull secrets attached.')).toBeVisible({ timeout: 15000 });
    await argosScreenshot(page, 'agent-hook-image-pull-secrets-empty', { fullPage: false });

    const secretLabel = `${registry} (${username})`;
    await attachSecret(page, secretLabel);

    const attachedRow = dialog.getByTestId('nested-image-pull-secret-row').filter({ hasText: registry });
    await expect(attachedRow).toBeVisible({ timeout: 15000 });
    await argosScreenshot(page, 'agent-hook-image-pull-secrets-attached', { fullPage: false });

    await attachedRow.getByTestId('nested-image-pull-secret-detach').click();
    await expect(attachedRow).toHaveCount(0, { timeout: 15000 });
    await expect(dialog.getByText('No image pull secrets attached.')).toBeVisible({ timeout: 15000 });
    await argosScreenshot(page, 'agent-hook-image-pull-secrets-detached', { fullPage: false });
  });
});
