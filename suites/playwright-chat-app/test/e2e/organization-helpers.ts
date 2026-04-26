import type { Page } from '@playwright/test';
import { listAccessibleOrganizations } from './chat-api';

async function waitForOrganization(page: Page, organizationId: string): Promise<void> {
  const timeoutMs = 10000;
  const start = Date.now();
  while (Date.now() - start < timeoutMs) {
    const organizations = await listAccessibleOrganizations(page);
    if (organizations.some((org) => org.id === organizationId)) {
      return;
    }
    await page.waitForTimeout(500);
  }
  throw new Error(`Organization ${organizationId} did not appear in time.`);
}

export async function setSelectedOrganization(page: Page, organizationId: string): Promise<void> {
  await waitForOrganization(page, organizationId);
  await page.evaluate((orgId) => {
    window.localStorage.setItem('ui.organization.selected', orgId);
  }, organizationId);
  await page.reload();
  await page.waitForSelector('[data-testid="chat-list"], [data-testid="no-organizations-screen"]', {
    timeout: 30000,
  });
}
