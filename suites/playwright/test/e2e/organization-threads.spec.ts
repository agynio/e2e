import { argosScreenshot } from '@argos-ci/playwright';
import { test, expect } from './fixtures';
import {
  createOrganization,
  createThread,
  createUser,
  getMe,
  sendThreadMessage,
  setSelectedOrganization,
} from './console-api';

test.describe('organization-threads', { tag: ['@svc_console', '@svc_gateway', '@svc_threads', '@svc_identity'] }, () => {
  test('org threads list and detail pagination', async ({ page }) => {
    const organizationId = await createOrganization(page, `e2e-org-threads-${Date.now()}`);
    await setSelectedOrganization(page, organizationId);
    const me = await getMe(page);
    const identityId = me.user?.meta?.id;
    if (!identityId) {
      throw new Error('GetMe response missing identity id for threads test.');
    }

    const participantId = await createUser(page, {
      email: `e2e-thread-participant-${Date.now()}@agyn.test`,
      nickname: 'thread-participant',
    });

    const threadOne = await createThread(page, { organizationId, participantIds: [participantId] });
    await sendThreadMessage(page, { threadId: threadOne, senderId: identityId, body: 'First thread message.' });

    const threadTwo = await createThread(page, { organizationId, participantIds: [participantId] });
    const totalMessages = 55;
    for (let index = 0; index < totalMessages; index += 1) {
      await sendThreadMessage(page, {
        threadId: threadTwo,
        senderId: identityId,
        body: `Thread message ${index + 1}`,
      });
    }

    await page.goto(`/organizations/${organizationId}/threads`);
    await expect(page.getByTestId('organization-threads-table')).toBeVisible({ timeout: 15000 });
    const rows = page.getByTestId('organization-thread-row');
    await expect(rows.first()).toContainText(threadTwo);
    await expect(rows.filter({ hasText: threadOne })).toBeVisible();
    await expect(rows.first().getByTestId('organization-thread-messages')).toHaveText('55');
    await argosScreenshot(page, 'organization-threads-list');

    await rows.first().click();
    await expect(page.getByTestId('thread-detail-card')).toBeVisible({ timeout: 15000 });
    await expect(page.getByTestId('thread-participants-card')).toBeVisible();
    await expect(page.getByTestId('thread-messages-card')).toBeVisible();

    await expect(page.getByTestId('thread-message-row')).toHaveCount(50);
    await expect(page.getByTestId('thread-message-row').first()).toContainText('Thread message 55');
    const loadMoreButton = page.getByTestId('thread-messages-card').getByTestId('load-more');
    await expect(loadMoreButton).toBeVisible();
    await expect(loadMoreButton).toHaveText('Load more');
    await argosScreenshot(page, 'organization-thread-detail');
    await loadMoreButton.click();
    await expect(page.getByTestId('thread-message-row')).toHaveCount(totalMessages);
    await expect(page.getByTestId('thread-messages-card').getByTestId('load-more')).toHaveCount(0);
  });
});
