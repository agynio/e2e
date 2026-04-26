import { argosScreenshot } from '@argos-ci/playwright';
import { fileURLToPath } from 'node:url';
import { test, expect } from './fixtures';
import {
  createAgentEnv,
  createChat,
  resolveIdentityId,
  sendFakeAgentReply,
  sendChatMessage,
  shouldUseFakeAgentReplies,
  setupTestAgent,
  waitForAgentReply,
} from './chat-api';
import { setSelectedOrganization } from './organization-helpers';

const defaultTestLlmEndpoint = 'https://testllm.dev/v1/org/agynio/suite/codex/responses';
const llmEndpoint = process.env.E2E_TEST_LLM_ENDPOINT ?? defaultTestLlmEndpoint;

test.describe('file-upload', {
  tag: [
    '@svc_chat_app',
    '@svc_gateway',
    '@svc_agents_orchestrator',
    '@svc_organizations',
    '@svc_files',
    '@svc_media_proxy',
  ],
}, () => {
  test('uploads a file and renders attachment', async ({ page }) => {
    test.setTimeout(180_000);
    const { organizationId, agentId, participantId } = await setupTestAgent(page, {
      endpoint: llmEndpoint,
      initImage: process.env.E2E_AGENT_INIT_IMAGE,
    });
    await createAgentEnv(page, agentId, 'TEST_SCENARIO', 'attachments');
    const identityId = await resolveIdentityId(page);
    const chatId = await createChat(page, organizationId, participantId);
    await setSelectedOrganization(page, organizationId);
    const useFakeAgent = shouldUseFakeAgentReplies();

    const chatLoaded = page.waitForResponse(
      (resp) => resp.url().includes('GetMessages') && resp.status() === 200,
      { timeout: 15000 },
    );
    await page.goto(`/chats/${encodeURIComponent(chatId)}`);
    await chatLoaded;

    const fixturePath = fileURLToPath(new URL('./fixtures/test-upload.txt', import.meta.url));
    const attachmentInput = page.getByTestId('file-attachment-input');
    await expect(attachmentInput).toBeAttached({ timeout: 15000 });
    await attachmentInput.setInputFiles(fixturePath);
    await expect(page.getByTestId('attachment-chip')).toBeVisible({ timeout: 15000 });

    const editor = page.getByTestId('markdown-composer-editor');
    await editor.click();
    await page.keyboard.type('Please summarize the file.');

    const sendResponse = page.waitForResponse(
      (resp) => resp.url().includes('SendMessage') && resp.status() === 200,
      { timeout: 15000 },
    );
    await page.getByLabel('Send message').click();
    await sendResponse;

    if (useFakeAgent) {
      await sendFakeAgentReply(page, chatId, 'Here is the attachment summary.');
    }

    await waitForAgentReply(page, chatId, identityId, 180000);
    await page.reload();

    const attachments = page.getByTestId('message-attachments');
    await expect(attachments.first()).toBeVisible({ timeout: 120000 });
    await argosScreenshot(page, 'chat-file-upload');
  });
});
