import type { Page } from '@playwright/test';
import { test as base, expect } from '@playwright/test';
import { signInViaOidc } from './sign-in-helper';

export { expect };

type TestFixtures = {};

async function signInAndLoad(page: Page) {
  await signInViaOidc(page);
}

export const test = base.extend<TestFixtures>({
  page: async ({ page }, runPage) => {
    page.on('console', (msg) => {
      if (msg.type() === 'error') {
        console.log('[browser-error]', msg.text());
      }
    });
    page.on('requestfailed', (request) => {
      console.log(`[request-failed] ${request.url()} — ${request.failure()?.errorText}`);
    });
    await signInAndLoad(page);
    await runPage(page);
  },
});
