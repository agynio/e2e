import { test, expect } from '@playwright/test';
import { readOidcSession } from './oidc-helpers';
import { clearAuthState, completeMockAuthLogin, ensureMockAuthEmailStrategy } from './sign-in-helper';
import { createFullChainRun } from './tracing-run';

test.describe('message deep link oidc callback', { tag: ['@svc_tracing_app', '@svc_agents_orchestrator'] }, () => {
  test.beforeAll(async ({ playwright }) => {
    const request = await playwright.request.newContext();
    try {
      await ensureMockAuthEmailStrategy(request);
    } finally {
      await request.dispose();
    }
  });

  test('returns to deep link after login', async ({ page }) => {
    test.setTimeout(8 * 60_000);

    await page.goto('/');
    const initialCallback = page.waitForURL(/\/callback/, { timeout: 60000 });
    await completeMockAuthLogin(page);
    await initialCallback;

    await expect
      .poll(async () => {
        const session = await readOidcSession(page);
        return session?.accessToken ?? '';
      }, { timeout: 60000 })
      .not.toBe('');

    const run = await createFullChainRun(page);

    await clearAuthState(page);

    const messageUrl = `/message/${run.messageId}?orgId=${run.organizationId}`;
    await page.goto(messageUrl);

    const callbackPromise = page.waitForURL(/\/callback/, { timeout: 60000 });
    await completeMockAuthLogin(page);
    await callbackPromise;

    const messageUrlPattern = new RegExp(`/message/${run.messageId}\\?orgId=${run.organizationId}$`);
    const runUrlPattern = new RegExp(`/${run.organizationId}/runs/${run.runId}(\\?.*)?$`);
    const finalUrlPattern = new RegExp(`${messageUrlPattern.source}|${runUrlPattern.source}`);
    await expect(page).toHaveURL(finalUrlPattern, { timeout: 60000 });

    const currentUrl = page.url();
    if (runUrlPattern.test(currentUrl)) {
      const runIdFromUrl = new URL(currentUrl).pathname.split('/').pop();
      if (!runIdFromUrl) {
        throw new Error(`Run redirect URL did not include a run id: ${currentUrl}`);
      }
      expect(runIdFromUrl).toBe(run.runId);
    } else {
      await expect(page).toHaveURL(messageUrlPattern);
    }
  });
});
