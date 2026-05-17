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
  const chatsLoaded = page.waitForResponse(
    (resp) => resp.url().includes('GetChats') && resp.status() === 200,
    { timeout: 30000 },
  );
  await page.reload();
  await page.waitForSelector('[data-testid="chat-list"], [data-testid="no-organizations-screen"]', {
    timeout: 30000,
  });

  // The app may render the chat list before React Query has finished the organization-scoped
  // chat request. Wait here, immediately after the reload that triggers the request, so tests
  // don't attach late GetChats listeners and race an already-loaded/cached route.
  await chatsLoaded.catch(async () => {
    const selectedOrganizationId = await page.evaluate(() => window.localStorage.getItem('ui.organization.selected'));
    if (selectedOrganizationId !== organizationId) {
      throw new Error(`Expected selected organization ${organizationId}, got ${selectedOrganizationId ?? '(none)'}.`);
    }
  });
}

export async function waitForChatList(page: Page, organizationId?: string): Promise<void> {
  await page.waitForSelector('[data-testid="chat-list"]', { timeout: 30000 });
  if (!organizationId) return;
  await page.waitForFunction(
    (orgId) => window.localStorage.getItem('ui.organization.selected') === orgId,
    organizationId,
    { timeout: 30000 },
  );
}
