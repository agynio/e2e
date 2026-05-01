import { test, expect } from './fixtures';
import {
  createOrganization,
  createThread,
  createUser,
  getMe,
  sendThreadMessage,
  setSelectedOrganization,
} from './console-api';

test.describe('organization-threads-smoke', {
  tag: ['@svc_console', '@svc_gateway', '@svc_threads', '@svc_identity', '@smoke'],
}, () => {
  test('threads list loads with data', async ({ page }) => {
    const organizationId = await createOrganization(page, `e2e-org-threads-smoke-${Date.now()}`);
    await setSelectedOrganization(page, organizationId);
    const me = await getMe(page);
    const identityId = me.user?.meta?.id;
    if (!identityId) {
      throw new Error('GetMe response missing identity id for threads smoke test.');
    }

    const participantId = await createUser(page, {
      email: `e2e-thread-smoke-${Date.now()}@agyn.test`,
      nickname: 'thread-smoke',
    });

    const threadId = await createThread(page, { organizationId, participantIds: [participantId] });
    await sendThreadMessage(page, { threadId, senderId: identityId, body: 'Smoke thread message.' });

    const threadsLoaded = page.waitForResponse(
      (resp) => resp.url().includes('ListOrganizationThreads'),
      { timeout: 20000 },
    );

    await page.goto(`/organizations/${organizationId}/threads`);
    const threadsResponse = await threadsLoaded;
    expect(threadsResponse.status()).toBe(200);
    await expect(page.getByTestId('organization-threads-table')).toBeVisible({ timeout: 20000 });
    const rows = page.getByTestId('organization-thread-row');
    const matchingRow = rows.filter({ hasText: threadId });
    await expect(matchingRow).toBeVisible({ timeout: 20000 });
    await expect(page.getByTestId('organization-threads-empty')).toHaveCount(0, { timeout: 20000 });
    await expect(page.getByText('Failed to load threads.')).toHaveCount(0);
    await expect(page.getByText('You do not have permission to view threads.')).toHaveCount(0);
  });
});
