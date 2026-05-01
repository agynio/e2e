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
    const statusHeader = header.getByRole('button', { name: 'Status' });
    await expect(agentHeader).toBeVisible();
    await expect(statusHeader).toBeVisible();

    const agentBox = await agentHeader.boundingBox();
    const statusBox = await statusHeader.boundingBox();
    if (!agentBox || !statusBox) {
      throw new Error('Workloads header bounding boxes missing.');
    }

    const rowOffset = Math.abs(agentBox.y - statusBox.y);
    expect(rowOffset).toBeLessThan(24);
    expect(statusBox.x).toBeGreaterThan(agentBox.x + 120);
  });
});
