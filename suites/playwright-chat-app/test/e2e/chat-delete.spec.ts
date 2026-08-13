import { test, expect } from './multi-user-fixtures';
import { createChat, createOrganization, listChats, resolveIdentityId, sendChatMessage } from './chat-api';
import { setSelectedOrganization, waitForChatList } from './organization-helpers';

test.describe('chat-delete', { tag: ['@svc_chat_app', '@svc_gateway', '@svc_organizations'] }, () => {
  test('deletes a conversation from the actions menu', async ({ userAPage, userBPage }) => {
    const now = Date.now();
    const message = `E2E delete message ${now}`;
    const organizationId = await createOrganization(userAPage, `e2e-org-delete-${now}`);
    const userBId = await resolveIdentityId(userBPage);
    const chatId = await createChat(userAPage, organizationId, userBId);
    await sendChatMessage(userAPage, chatId, message);
    await setSelectedOrganization(userAPage, organizationId);

    const messagesLoaded = userAPage.waitForResponse(
      (resp) => resp.url().includes('GetMessages') && resp.status() === 200,
      { timeout: 15000 },
    );
    await userAPage.goto(`/chats/${encodeURIComponent(chatId)}`);
    await waitForChatList(userAPage, organizationId);
    await messagesLoaded;

    await userAPage.getByTestId('chat-detail-header-menu').click();
    await userAPage.getByTestId('chat-action-delete').click();
    await expect(userAPage.getByTestId('chat-delete-dialog')).toBeVisible();

    const deleted = userAPage.waitForResponse(
      (resp) => resp.url().includes('DeleteChat') && resp.status() === 200,
      { timeout: 15000 },
    );
    await userAPage.getByTestId('chat-delete-confirm').click();
    await deleted;

    await expect(userAPage).toHaveURL(/\/chats\/?$/, { timeout: 15000 });
    await expect(userAPage.getByTestId('chat-detail-header-menu')).toHaveCount(0);

    await expect
      .poll(async () => (await listChats(userAPage, organizationId)).some((chat) => chat.id === chatId), {
        timeout: 15000,
      })
      .toBe(false);
  });
});
