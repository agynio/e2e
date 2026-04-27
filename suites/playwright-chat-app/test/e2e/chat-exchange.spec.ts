import { argosScreenshot } from '@argos-ci/playwright';
import { test, expect } from './multi-user-fixtures';
import {
  acceptMembership,
  createChat,
  createMembership,
  createOrganization,
  resolveIdentityId,
  sendChatMessage,
} from './chat-api';
import { setSelectedOrganization } from './organization-helpers';

test.describe('chat-exchange', { tag: ['@svc_chat_app', '@svc_gateway', '@svc_organizations'] }, () => {
  test('two users exchange messages in a shared chat', async ({ userAPage, userBPage }) => {
    const messageFromA = `Hello from User A ${Date.now()}`;
    const messageFromB = `Reply from User B ${Date.now()}`;

    const userBId = await resolveIdentityId(userBPage);
    const organizationId = await createOrganization(userAPage, `e2e-org-exchange-a-${Date.now()}`);
    const chatId = await createChat(userAPage, organizationId, userBId);
    await sendChatMessage(userAPage, chatId, messageFromA);
    await setSelectedOrganization(userAPage, organizationId);
    // User B needs their own org to pass the org gate; direct URL access works because GetMessages/SendMessage ignore orgs.
    const userBOrganizationId = await createOrganization(userBPage, `e2e-org-exchange-b-${Date.now()}`);
    await setSelectedOrganization(userBPage, userBOrganizationId);

    const userAChatsLoaded = userAPage.waitForResponse(
      (resp) => resp.url().includes('GetChats') && resp.status() === 200,
      { timeout: 15000 },
    );
    const userAMessagesLoaded = userAPage.waitForResponse(
      (resp) => resp.url().includes('GetMessages') && resp.status() === 200,
      { timeout: 15000 },
    );
    await userAPage.goto(`/chats/${encodeURIComponent(chatId)}`);
    await userAChatsLoaded;
    await userAMessagesLoaded;
    await expect(userAPage.getByTestId('chat-message').filter({ hasText: messageFromA })).toBeVisible({
      timeout: 15000,
    });

    const userBChatsLoaded = userBPage.waitForResponse(
      (resp) => resp.url().includes('GetChats') && resp.status() === 200,
      { timeout: 15000 },
    );
    const userBMessagesLoaded = userBPage.waitForResponse(
      (resp) => resp.url().includes('GetMessages') && resp.status() === 200,
      { timeout: 15000 },
    );
    await userBPage.goto(`/chats/${encodeURIComponent(chatId)}`);
    await userBChatsLoaded;
    await userBMessagesLoaded;
    await expect(userBPage.getByTestId('chat-message').filter({ hasText: messageFromA })).toBeVisible({
      timeout: 15000,
    });

    const editorB = userBPage.getByTestId('markdown-composer-editor');
    await editorB.click();
    await userBPage.keyboard.type(messageFromB);
    const userBSendMessage = userBPage.waitForResponse(
      (resp) => resp.url().includes('SendMessage') && resp.status() === 200,
      { timeout: 15000 },
    );
    await userBPage.getByLabel('Send message').click();
    await userBSendMessage;
    await expect(userBPage.getByTestId('chat-message').filter({ hasText: messageFromB })).toBeVisible({
      timeout: 15000,
    });
    const userAMessagesRefreshed = userAPage.waitForResponse(
      (resp) => resp.url().includes('GetMessages') && resp.status() === 200,
      { timeout: 15000 },
    );
    await userAPage.reload();
    await userAMessagesRefreshed;
    await expect(userAPage.getByTestId('chat-message').filter({ hasText: messageFromB })).toBeVisible({
      timeout: 15000,
    });
    await argosScreenshot(userAPage, 'two-user-message-exchange');
  });

  test('user B sees shared chat in their chat list', async ({ userAPage, userBPage }) => {
    const messageFromA = `Hello from User A ${Date.now()}`;

    const userBId = await resolveIdentityId(userBPage);
    const organizationId = await createOrganization(userAPage, `e2e-org-exchange-${Date.now()}`);
    // Add User B to the organization
    const membershipId = await createMembership(userAPage, organizationId, userBId);
    // Accept in case the invite is still pending.
    await acceptMembership(userBPage, membershipId);
    const chatId = await createChat(userAPage, organizationId, userBId);
    await sendChatMessage(userAPage, chatId, messageFromA);
    await setSelectedOrganization(userBPage, organizationId);

    await userBPage.goto('/chats');

    const chatList = userBPage.getByTestId('chat-list');
    await expect(chatList).toBeVisible({ timeout: 15000 });

    const firstChat = chatList.locator('.cursor-pointer').first();
    await expect(firstChat).toBeVisible({ timeout: 15000 });

    const messagesLoaded = userBPage.waitForResponse(
      (resp) => resp.url().includes('GetMessages') && resp.status() === 200,
      { timeout: 15000 },
    );
    await firstChat.click();
    await messagesLoaded;

    await expect(userBPage).toHaveURL(/\/chats\/.+/, { timeout: 15000 });
    await expect(userBPage.getByTestId('chat-message').filter({ hasText: messageFromA })).toBeVisible({
      timeout: 15000,
    });
    await argosScreenshot(userBPage, 'user-b-sees-shared-chat');
  });
});
