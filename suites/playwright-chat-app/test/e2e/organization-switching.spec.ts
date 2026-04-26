import { argosScreenshot } from '@argos-ci/playwright';
import type { Page } from '@playwright/test';
import { test, expect } from './fixtures';
import {
  createAgent,
  createOrganization,
  createTestModel,
  DEFAULT_TEST_AGENT_IMAGE,
  DEFAULT_TEST_INIT_IMAGE,
} from './chat-api';
import { setSelectedOrganization } from './organization-helpers';

const defaultTestLlmEndpoint = 'https://testllm.dev/v1/org/agynio/suite/codex/responses';
const llmEndpoint = process.env.E2E_TEST_LLM_ENDPOINT ?? defaultTestLlmEndpoint;

async function createAgentForOrg(page: Page, organizationId: string, name: string) {
  const { modelId } = await createTestModel(page, {
    organizationId,
    endpoint: llmEndpoint,
    namePrefix: 'e2e-model-org-switch',
  });
  await createAgent(page, {
    organizationId,
    name,
    role: 'assistant',
    model: modelId,
    description: 'Org switch agent',
    configuration: '{}',
    image: DEFAULT_TEST_AGENT_IMAGE,
    initImage: DEFAULT_TEST_INIT_IMAGE,
  });
}

test.describe('organization-switching', {
  tag: ['@svc_chat_app', '@svc_gateway', '@svc_agents_orchestrator', '@svc_organizations'],
}, () => {
  test('switching organization updates chat list', async ({ page }) => {
    const now = Date.now();
    const organizationNameA = `e2e-org-a-${now}`;
    const organizationNameB = `e2e-org-b-${now}`;
    const organizationIdA = await createOrganization(page, organizationNameA);
    const organizationIdB = await createOrganization(page, organizationNameB);

    await createAgentForOrg(page, organizationIdA, `agent-a-${now}`);
    await createAgentForOrg(page, organizationIdB, `agent-b-${now}`);

    await setSelectedOrganization(page, organizationIdA);
    await page.goto('/chats');
    const list = page.getByTestId('chat-list');
    await expect(list).toBeVisible({ timeout: 15000 });
    await argosScreenshot(page, 'org-switch-org-a');

    const userMenuTrigger = page.getByTestId('user-menu-trigger');
    await expect(userMenuTrigger).toBeVisible({ timeout: 15000 });
    await userMenuTrigger.click();

    const orgSwitcher = page.getByTestId('org-switcher');
    await expect(orgSwitcher).toBeVisible({ timeout: 15000 });
    await orgSwitcher.click();

    const orgItem = page.getByTestId(`org-item-${organizationIdB}`);
    await expect(orgItem).toBeVisible({ timeout: 15000 });
    await orgItem.click({ force: true });

    await expect(list).toBeVisible({ timeout: 15000 });
    await argosScreenshot(page, 'org-switch-org-b');
  });
});
