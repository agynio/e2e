import { test, expect } from './fixtures';
import { registerRunner } from './console-api';

test.describe('workloads-layout', { tag: ['@svc_console', '@svc_gateway', '@smoke'] }, () => {
  test('workloads header lays out across columns', async ({ page }) => {
    const runnerName = `e2e-workloads-layout-${Date.now()}`;
    const runner = await registerRunner(page, { name: runnerName, labels: { scope: 'cluster' } });
    const runnerId = runner.meta?.id;
    if (!runnerId) {
      throw new Error('RegisterRunner response missing runner id for workloads layout test.');
    }

    await page.goto(`/runners/${runnerId}`);
    const header = page.getByTestId('runner-workloads-header');
    await expect(header).toBeVisible({ timeout: 15000 });

    const agentHeader = header.getByText('Agent', { exact: true });
    const actionHeader = header.getByText('Action', { exact: true });
    await expect(agentHeader).toBeVisible();
    await expect(actionHeader).toBeVisible();

    const agentBox = await agentHeader.boundingBox();
    const actionBox = await actionHeader.boundingBox();
    if (!agentBox || !actionBox) {
      throw new Error('Workloads header bounding boxes missing.');
    }

    const rowOffset = Math.abs(agentBox.y - actionBox.y);
    expect(rowOffset).toBeLessThan(4);
    expect(actionBox.x).toBeGreaterThan(agentBox.x + 100);
  });
});
