import { argosScreenshot } from '@argos-ci/playwright';
import type { Page } from '@playwright/test';
import { test, expect } from './fixtures';
import { createAgent, createOrganization, DEFAULT_TEST_INIT_IMAGE } from './chat-api';

async function waitForOrganization(page: Page, organizationId: string): Promise<void> {
  const timeoutMs = 10000;
  const start = Date.now();
  while (Date.now() - start < timeoutMs) {
    const response = await page.request.post('/api/agynio.api.gateway.v1.OrganizationsGateway/ListAccessibleOrganizations', {
      data: {},
      headers: {
        'Content-Type': 'application/json',
        'Connect-Protocol-Version': '1',
      },
    });
    if (!response.ok()) {
      throw new Error(`Failed to list organizations: ${response.status()}`);
    }
    const { organizations } = (await response.json()) as { organizations?: Array<{ id: string }> };
    if (organizations?.some((org) => org.id === organizationId)) {
      return;
    }
    await page.waitForTimeout(500);
  }
  throw new Error(`Organization ${organizationId} did not appear in time.`);
}

async function setSelectedOrganization(page: Page, organizationId: string): Promise<void> {
  await waitForOrganization(page, organizationId);
  await page.evaluate((orgId) => {
    window.localStorage.setItem('ui.organization.selected', orgId);
  }, organizationId);
  await page.reload();
}

async function createAgentForOrg(page: Page, organizationId: string, name: string) {
  await createAgent(page, {
    organizationId,
    name,
    role: 'assistant',
    model: 'model-id',
    description: 'Org switch agent',
    configuration: '{}',
    image: 'agent-image:latest',
    initImage: DEFAULT_TEST_INIT_IMAGE,
  });
}

test.describe('organization-switching', { tag: ['@svc_chat_app', '@svc_gateway', '@svc_agents_orchestrator'] }, () => {
  test('switching organization updates chat list', async ({ page }) => {
    const now = Date.now();
    const organizationIdA = await createOrganization(page, `e2e-org-a-${now}`);
    const organizationIdB = await createOrganization(page, `e2e-org-b-${now}`);

    await createAgentForOrg(page, organizationIdA, `agent-a-${now}`);
    await createAgentForOrg(page, organizationIdB, `agent-b-${now}`);

    await setSelectedOrganization(page, organizationIdA);
    await page.goto('/chats');
    const list = page.getByTestId('chat-list');
    await expect(list).toBeVisible({ timeout: 15000 });
    await argosScreenshot(page, 'org-switch-org-a');

    const orgSelector = page.getByTestId('organization-switcher');
    await expect(orgSelector).toBeVisible({ timeout: 15000 });
    await orgSelector.click();
    await page.getByRole('option', { name: organizationIdB }).click();

    await expect(list).toBeVisible({ timeout: 15000 });
    await argosScreenshot(page, 'org-switch-org-b');
  });
});
