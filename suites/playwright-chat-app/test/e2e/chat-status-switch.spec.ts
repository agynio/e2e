import { argosScreenshot } from '@argos-ci/playwright';
import { test, expect } from './multi-user-fixtures';
import { createChat, createOrganization, resolveIdentityId, updateChatStatus } from './chat-api';
import { setSelectedOrganization } from './organization-helpers';

test.describe('chat-status-switch', { tag: ['@svc_chat_app', '@svc_gateway', '@svc_organizations'] }, () => {
  test('switches from open to closed chat status', async ({ userAPage, userBPage }) => {
    const organizationId = await createOrganization(userAPage, `e2e-org-status-${Date.now()}`);
    const userBId = await resolveIdentityId(userBPage);
    const chatId = await createChat(userAPage, organizationId, userBId);
    await updateChatStatus(userAPage, chatId, 'closed');
    await setSelectedOrganization(userAPage, organizationId);

    const messagesLoaded = userAPage.waitForResponse(
      (resp) => resp.url().includes('GetMessages') && resp.status() === 200,
      { timeout: 15000 },
    );
    await userAPage.goto(`/chats/${encodeURIComponent(chatId)}`);
    await messagesLoaded;

    await expect(userAPage.getByRole('button', { name: 'Chat status: Resolved' })).toBeVisible({ timeout: 15000 });
    await argosScreenshot(userAPage, 'chat-status-closed');
  });
});
