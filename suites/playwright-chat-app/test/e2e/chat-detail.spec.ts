import { argosScreenshot } from '@argos-ci/playwright';
import { test, expect } from './multi-user-fixtures';
import { createChat, createOrganization, resolveIdentityId, sendChatMessage } from './chat-api';
import { setSelectedOrganization } from './organization-helpers';

test.describe('chat-detail', { tag: ['@svc_chat_app', '@svc_gateway', '@svc_organizations'] }, () => {
  test('shows chat messages', async ({ userAPage, userBPage }) => {
    const now = Date.now();
    const message = `E2E detail message ${now}`;
    const organizationId = await createOrganization(userAPage, `e2e-org-detail-${now}`);
    const userBId = await resolveIdentityId(userBPage);
    const chatId = await createChat(userAPage, organizationId, userBId);
    await sendChatMessage(userAPage, chatId, message);
    await setSelectedOrganization(userAPage, organizationId);

    const messagesLoaded = userAPage.waitForResponse(
      (resp) => resp.url().includes('GetMessages') && resp.status() === 200,
      { timeout: 15000 },
    );
    await userAPage.goto(`/chats/${encodeURIComponent(chatId)}`);
    await messagesLoaded;

    const messageItem = userAPage.getByTestId('chat-message').filter({ hasText: message });
    await expect(messageItem).toBeVisible({ timeout: 15000 });
    await argosScreenshot(userAPage, 'chat-detail-messages');
  });
});
