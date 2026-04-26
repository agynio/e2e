import { argosScreenshot } from '@argos-ci/playwright';
import { test, expect } from './fixtures';
import {
  createChat,
  resolveIdentityId,
  sendChatMessage,
  sendFakeAgentReply,
  setupTestAgent,
  shouldUseFakeAgentReplies,
  waitForAgentReply,
} from './chat-api';
import { setSelectedOrganization } from './organization-helpers';

const defaultTestLlmEndpoint = 'https://testllm.dev/v1/org/agynio/suite/codex/responses';
const llmEndpoint = process.env.E2E_TEST_LLM_ENDPOINT ?? defaultTestLlmEndpoint;

test.describe('chat-with-agent', {
  tag: ['@svc_chat_app', '@svc_gateway', '@svc_agents_orchestrator', '@svc_organizations'],
}, () => {
  test('chat with agent and receive reply', async ({ page }) => {
    test.setTimeout(180_000);
    const { organizationId, participantId } = await setupTestAgent(page, {
      endpoint: llmEndpoint,
      initImage: process.env.E2E_AGENT_INIT_IMAGE,
    });
    const identityId = await resolveIdentityId(page);
    const chatId = await createChat(page, organizationId, participantId);
    await setSelectedOrganization(page, organizationId);
    const useFakeAgent = shouldUseFakeAgentReplies();
    const message = `Hello agent ${Date.now()}`;
    await sendChatMessage(page, chatId, message);
    if (useFakeAgent) {
      await sendFakeAgentReply(page, chatId, `Agent reply ${Date.now()}`);
    }

    const chatLoaded = page.waitForResponse(
      (resp) => resp.url().includes('GetMessages') && resp.status() === 200,
      { timeout: 15000 },
    );
    await page.goto(`/chats/${encodeURIComponent(chatId)}`);
    await chatLoaded;

    await waitForAgentReply(page, chatId, identityId, 180_000);
    await page.reload();

    const messageList = page.getByTestId('chat-message');
    await expect(messageList).toHaveCount(2, { timeout: 180000 });
    const agentMessage = messageList.filter({ hasNotText: 'Hello agent' }).first();
    await expect(agentMessage).toBeVisible({ timeout: 180000 });

    await argosScreenshot(page, 'chat-agent-reply');
  });
});
