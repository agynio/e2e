import { argosScreenshot } from '@argos-ci/playwright';
import { test, expect } from './fixtures';
import { createChat, resolveIdentityId, sendChatMessage, setupTestAgent } from './chat-api';
import { setSelectedOrganization } from './organization-helpers';

const defaultTestLlmEndpoint = 'https://test-llm.agyn.dev';
const llmEndpoint = process.env.E2E_TEST_LLM_ENDPOINT ?? defaultTestLlmEndpoint;

test.describe('chat-with-agent', { tag: ['@svc_chat_app', '@svc_gateway', '@svc_agents_orchestrator'] }, () => {
  test('chat with agent and receive reply', async ({ page }) => {
    test.setTimeout(120_000);
    const { organizationId, agentId } = await setupTestAgent(page, {
      endpoint: llmEndpoint,
      initImage: process.env.E2E_AGENT_INIT_IMAGE,
    });
    const identityId = await resolveIdentityId(page);
    const chatId = await createChat(page, organizationId, identityId);
    await setSelectedOrganization(page, organizationId);
    await sendChatMessage(page, chatId, `Hello agent ${Date.now()}`);

    const chatLoaded = page.waitForResponse(
      (resp) => resp.url().includes('GetMessages') && resp.status() === 200,
      { timeout: 15000 },
    );
    await page.goto(`/chats/${encodeURIComponent(chatId)}`);
    await chatLoaded;

    const messageList = page.getByTestId('chat-message');
    await expect(messageList).toHaveCount(2, { timeout: 120000 });
    const agentMessage = messageList.filter({ hasNotText: 'Hello agent' }).first();
    await expect(agentMessage).toBeVisible({ timeout: 120000 });

    await argosScreenshot(page, 'chat-agent-reply');

    await page.getByTestId('chat-details').click();
    const assignedAgents = page.getByTestId('chat-details-agents');
    await expect(assignedAgents).toContainText(agentId, { timeout: 15000 });
  });

  test('assigning agent shows in chat details', async ({ page }) => {
    test.setTimeout(120_000);
    const { organizationId, agentId, agentName } = await setupTestAgent(page, {
      endpoint: llmEndpoint,
      initImage: process.env.E2E_AGENT_INIT_IMAGE,
    });
    const identityId = await resolveIdentityId(page);
    const chatId = await createChat(page, organizationId, identityId);
    await setSelectedOrganization(page, organizationId);

    const chatLoaded = page.waitForResponse(
      (resp) => resp.url().includes('GetMessages') && resp.status() === 200,
      { timeout: 15000 },
    );
    await page.goto(`/chats/${encodeURIComponent(chatId)}`);
    await chatLoaded;

    const membersTab = page.getByRole('tab', { name: 'Members' });
    await expect(membersTab).toBeVisible({ timeout: 15000 });
    await membersTab.click();

    const addMemberButton = page.getByRole('button', { name: 'Add member' });
    await expect(addMemberButton).toBeVisible({ timeout: 15000 });
    await addMemberButton.click();

    const searchInput = page.getByPlaceholder('Search agents');
    await expect(searchInput).toBeVisible({ timeout: 15000 });
    await searchInput.click();

    const agentOption = page.getByRole('option', { name: agentName });
    await expect(agentOption).toBeVisible({ timeout: 15000 });
    await agentOption.click();

    const assignResponse = page.waitForResponse(
      (resp) => resp.url().includes('AssignAgents') && resp.status() === 200,
      { timeout: 15000 },
    );
    await page.getByRole('button', { name: 'Add member' }).click();
    await assignResponse;

    await page.getByTestId('chat-details').click();
    const assignedAgents = page.getByTestId('chat-details-agents');
    await expect(assignedAgents).toContainText(agentId, { timeout: 15000 });
  });
});
