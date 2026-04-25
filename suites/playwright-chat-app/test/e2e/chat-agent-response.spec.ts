import { argosScreenshot } from '@argos-ci/playwright';
import { test, expect } from './fixtures';
import {
  createAgentEnv,
  createChat,
  resolveIdentityId,
  sendChatMessage,
  setupTestAgent,
  waitForAgentReply,
} from './chat-api';
import { setSelectedOrganization } from './organization-helpers';

const defaultTestLlmEndpoint = 'https://test-llm.agyn.dev';
const llmEndpoint = process.env.E2E_TEST_LLM_ENDPOINT ?? defaultTestLlmEndpoint;

test.describe('chat-agent-response', { tag: ['@svc_chat_app', '@svc_gateway', '@svc_agents_orchestrator'] }, () => {
  test('agent response appears after message send', async ({ page }) => {
    test.setTimeout(120_000);
    const { organizationId, agentId } = await setupTestAgent(page, {
      endpoint: llmEndpoint,
      initImage: process.env.E2E_AGENT_INIT_IMAGE,
    });
    await createAgentEnv(page, agentId, 'TEST_SCENARIO', 'attachments');
    const identityId = await resolveIdentityId(page);
    const chatId = await createChat(page, organizationId, identityId);
    await setSelectedOrganization(page, organizationId);

    const message = `Hello agent response ${Date.now()}`;
    await sendChatMessage(page, chatId, message);

    const chatLoaded = page.waitForResponse(
      (resp) => resp.url().includes('GetMessages') && resp.status() === 200,
      { timeout: 15000 },
    );
    await page.goto(`/chats/${encodeURIComponent(chatId)}`);
    await chatLoaded;

    await waitForAgentReply(page, chatId, identityId);
    await page.reload();

    const messageList = page.getByTestId('chat-message');
    await expect(messageList).toHaveCount(2, { timeout: 120000 });
    await argosScreenshot(page, 'chat-agent-response');
  });
});
