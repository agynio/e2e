import { randomUUID } from 'node:crypto';
import { test, expect } from './fixtures';
import { createOrganization } from './gateway-api';

test.describe('message deep link empty state', { tag: ['@svc_tracing_app', '@smoke'] }, () => {
  test('shows empty state for unknown message', async ({ page }) => {
    const organizationId = await createOrganization(page, `e2e-empty-${randomUUID()}`);
    const messageId = randomUUID();

    await page.goto(`/message/${messageId}?orgId=${organizationId}`);

    await expect(page.getByText('No run found for message.')).toBeVisible({ timeout: 30000 });
    await expect(page).toHaveURL(new RegExp(`/message/${messageId}\\?orgId=${organizationId}`));
  });
});
