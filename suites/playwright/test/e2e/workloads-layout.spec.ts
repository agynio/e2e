import { test, expect } from './fixtures';
import { createOrganization, setSelectedOrganization } from './console-api';

test.describe('workloads-layout', { tag: ['@svc_console', '@svc_gateway', '@smoke'] }, () => {
  test('workloads header lays out across columns', async ({ page }) => {
    const organizationId = await createOrganization(page, `e2e-workloads-layout-${Date.now()}`);
    await setSelectedOrganization(page, organizationId);

    await page.goto(`/organizations/${organizationId}/activity/workloads`);
    const header = page.getByTestId('organization-workloads-header');
    await expect(header).toBeVisible({ timeout: 15000 });
    await header.scrollIntoViewIfNeeded();

    const agentHeader = header.getByRole('button', { name: 'Agent' });
    const durationHeader = header.getByRole('button', { name: 'Duration' });
    await expect(agentHeader).toBeVisible();
    await expect(durationHeader).toBeVisible();

    const agentBox = await agentHeader.boundingBox();
    const durationBox = await durationHeader.boundingBox();
    if (!agentBox || !durationBox) {
      throw new Error('Workloads header bounding boxes missing.');
    }

    const rowOffset = Math.abs(agentBox.y - durationBox.y);
    expect(rowOffset).toBeLessThan(8);
    expect(durationBox.x).toBeGreaterThan(agentBox.x + 150);
  });
});
