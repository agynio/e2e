import { argosScreenshot } from '@argos-ci/playwright';
import type { Page } from '@playwright/test';
import { test, expect } from './multi-user-fixtures';
import {
  createAgent,
  createChat,
  createOrganization,
  createTestModel,
  DEFAULT_TEST_AGENT_IMAGE,
  DEFAULT_TEST_INIT_IMAGE,
  resolveIdentityId,
} from './chat-api';
import { setSelectedOrganization } from './organization-helpers';

const defaultTestLlmEndpoint = 'https://testllm.dev/v1/org/agynio/suite/codex/responses';
const llmEndpoint = process.env.E2E_TEST_LLM_ENDPOINT ?? defaultTestLlmEndpoint;

async function expectChatListVisible(page: Page) {
  const list = page.getByTestId('chat-list');
  await expect(list).toBeVisible({ timeout: 15000 });
}

test.describe('chats-list', { tag: ['@svc_chat_app', '@svc_gateway', '@svc_organizations'] }, () => {
  test('renders chat list on load', async ({ userAPage }) => {
    const organizationId = await createOrganization(userAPage, `e2e-org-list-${Date.now()}`);
    await setSelectedOrganization(userAPage, organizationId);
    const chatsLoaded = userAPage.waitForResponse(
      (resp) => resp.url().includes('GetChats') && resp.status() === 200,
      { timeout: 15000 },
    );
    await userAPage.goto('/chats');
    await chatsLoaded;

    await expectChatListVisible(userAPage);
    await argosScreenshot(userAPage, 'chats-list-loaded');
  });

  test(
    'participant picker shows available options',
    { tag: ['@svc_chat_app', '@svc_gateway', '@svc_organizations', '@svc_agents_orchestrator'] },
    async ({ userAPage }) => {
      const now = Date.now();
      const organizationId = await createOrganization(userAPage, `e2e-org-picker-${now}`);
      await setSelectedOrganization(userAPage, organizationId);
      const { modelId } = await createTestModel(userAPage, {
        organizationId,
        endpoint: llmEndpoint,
        namePrefix: 'e2e-model-picker',
      });
      const agentName = `e2e-agent-picker-${now}`;
      await createAgent(userAPage, {
        organizationId,
        name: agentName,
        role: 'assistant',
        model: modelId,
        description: 'E2E participant picker agent',
        configuration: '{}',
        image: DEFAULT_TEST_AGENT_IMAGE,
        initImage: DEFAULT_TEST_INIT_IMAGE,
      });
      const chatsLoaded = userAPage.waitForResponse(
        (resp) => resp.url().includes('GetChats') && resp.status() === 200,
        { timeout: 15000 },
      );
      await userAPage.goto('/chats');
      await chatsLoaded;

      await expectChatListVisible(userAPage);

      const newChatBtn = userAPage.getByTitle('New chat');
      await expect(newChatBtn).toBeVisible({ timeout: 15000 });
      await newChatBtn.click();

      const autocomplete = userAPage.getByPlaceholder('Search participants...');
      await expect(autocomplete).toBeVisible({ timeout: 15000 });
      await autocomplete.click();

      await expect(userAPage.getByRole('option', { name: agentName })).toBeVisible({ timeout: 15000 });

      await argosScreenshot(userAPage, 'participant-picker-dropdown');
    },
  );

  test('redirects root to /chats', async ({ userAPage }) => {
    await userAPage.goto('/');

    await expect(userAPage).toHaveURL(/\/chats$/);
  });

  test('navigates to chat detail', async ({ userAPage, userBPage }) => {
    const now = Date.now();
    const organizationId = await createOrganization(userAPage, `e2e-org-detail-${now}`);
    const userBId = await resolveIdentityId(userBPage);
    const chatId = await createChat(userAPage, organizationId, userBId);
    await setSelectedOrganization(userAPage, organizationId);

    const chatsLoaded = userAPage.waitForResponse(
      (resp) => resp.url().includes('GetChats') && resp.status() === 200,
      { timeout: 15000 },
    );
    await userAPage.goto('/chats');
    await chatsLoaded;

    const chatList = userAPage.getByTestId('chat-list');
    await expect(chatList).toBeVisible({ timeout: 15000 });

    const firstChat = chatList.locator('.cursor-pointer').first();
    await expect(firstChat).toBeVisible({ timeout: 15000 });
    await firstChat.click();

    await expect(userAPage).toHaveURL(new RegExp(`/chats/${encodeURIComponent(chatId)}`));
    await expect(userAPage.getByTestId('chat')).toBeVisible({ timeout: 15000 });
    await argosScreenshot(userAPage, 'chats-list-detail');
  });
});
