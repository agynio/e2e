import { test, expect } from '@playwright/test';
import { readOidcSession } from './oidc-helpers';
import { clearAuthState, completeOidcLogin, signInViaOidc } from './sign-in-helper';
import { createFullChainRun } from './tracing-run';

test.describe('message deep link oidc callback', { tag: ['@svc_tracing_app', '@svc_agents_orchestrator'] }, () => {
  test('returns to deep link after login', async ({ page }) => {
    test.setTimeout(8 * 60_000);

    await signInViaOidc(page);

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

    const callbackPromise = page.waitForURL(/\/callback/, { timeout: 60000 }).catch(() => null);
    const completed = await completeOidcLogin(page);
    if (completed) {
      await callbackPromise;
    }

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
