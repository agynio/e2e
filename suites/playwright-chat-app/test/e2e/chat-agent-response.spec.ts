import { argosScreenshot } from '@argos-ci/playwright';
import { test, expect } from './fixtures';
import { ensureAgentReply } from './agent-reply-helper';
import { createChat, setupTestAgent } from './chat-api';
import { setSelectedOrganization } from './organization-helpers';

const defaultTestLlmEndpoint = 'https://testllm.dev/v1/org/agynio/suite/codex/responses';
const llmEndpoint = process.env.E2E_TEST_LLM_ENDPOINT ?? defaultTestLlmEndpoint;

test.describe('chat-agent-response', {
  tag: ['@svc_chat_app', '@svc_gateway', '@svc_agents_orchestrator', '@svc_organizations'],
}, () => {
  test('agent response appears after message send', async ({ page }) => {
    test.setTimeout(180_000);
    const { organizationId, participantId } = await setupTestAgent(page, {
      endpoint: llmEndpoint,
      initImage: process.env.E2E_AGENT_INIT_IMAGE,
    });
    const chatId = await createChat(page, organizationId, participantId);
    await setSelectedOrganization(page, organizationId);
    const message = 'hello';

    const chatLoaded = page.waitForResponse(
      (resp) => resp.url().includes('GetMessages') && resp.status() === 200,
      { timeout: 15000 },
    );
    await page.goto(`/chats/${encodeURIComponent(chatId)}`);
    await chatLoaded;

    const sendResponse = page.waitForResponse(
      (resp) => resp.url().includes('SendMessage') && resp.status() === 200,
      { timeout: 15000 },
    );
    await page.getByTestId('markdown-composer-editor').fill(message);
    await page.getByLabel('Send message').click();
    await sendResponse;

    await ensureAgentReply(page, chatId, message);

    const messageList = page.getByTestId('chat-message');
    await expect(messageList.first()).toContainText(message, { timeout: 180000 });
    await expect(messageList.nth(1)).toBeVisible({ timeout: 180000 });
    await argosScreenshot(page, 'chat-agent-response');
  });
});
