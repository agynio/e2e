import { argosScreenshot } from '@argos-ci/playwright';
import path from 'node:path';
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

test.describe('file-upload', { tag: ['@svc_chat_app', '@svc_gateway', '@svc_agents_orchestrator'] }, () => {
  test('uploads a file and renders attachment', async ({ page }) => {
    test.setTimeout(120_000);
    const { organizationId, agentId } = await setupTestAgent(page, {
      endpoint: llmEndpoint,
      initImage: process.env.E2E_AGENT_INIT_IMAGE,
    });
    await createAgentEnv(page, agentId, 'TEST_SCENARIO', 'attachments');
    const identityId = await resolveIdentityId(page);
    const chatId = await createChat(page, organizationId, identityId);
    await setSelectedOrganization(page, organizationId);

    const chatLoaded = page.waitForResponse(
      (resp) => resp.url().includes('GetMessages') && resp.status() === 200,
      { timeout: 15000 },
    );
    await page.goto(`/chats/${encodeURIComponent(chatId)}`);
    await chatLoaded;

    const fileChooserPromise = page.waitForEvent('filechooser');
    await page.getByLabel('Attach file').click();
    const fileChooser = await fileChooserPromise;
    const fixturePath = path.join(__dirname, 'fixtures', 'test-upload.txt');
    await fileChooser.setFiles(fixturePath);

    const editor = page.getByTestId('markdown-composer-editor');
    await editor.click();
    await page.keyboard.type('Please summarize the file.');

    const sendResponse = page.waitForResponse(
      (resp) => resp.url().includes('SendMessage') && resp.status() === 200,
      { timeout: 15000 },
    );
    await page.getByLabel('Send message').click();
    await sendResponse;

    await waitForAgentReply(page, chatId, identityId);
    await page.reload();

    const attachments = page.getByTestId('chat-message-attachment');
    await expect(attachments.first()).toBeVisible({ timeout: 120000 });
    await argosScreenshot(page, 'chat-file-upload');
  });
});
