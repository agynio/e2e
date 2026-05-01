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

    await page.goto(`/organizations/${organizationId}/threads`);
    await expect(page.getByTestId('organization-threads-table')).toBeVisible({ timeout: 15000 });
    const row = page.getByTestId('organization-thread-row').first();
    await expect(row).toBeVisible({ timeout: 15000 });
    await expect(row).toContainText(threadId);
    await expect(page.getByTestId('organization-threads-empty')).toHaveCount(0);
    await expect(page.getByText('Failed to load threads.')).toHaveCount(0);
    await expect(page.getByText('You do not have permission to view threads.')).toHaveCount(0);
  });
});
