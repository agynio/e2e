import type { Page } from '@playwright/test';
import { test as base, expect } from '@playwright/test';
import { ensureMockAuthEmailStrategy, seedOidcSessionViaMockAuth } from './sign-in-helper';

export { expect };

type TestFixtures = object;

type WorkerFixtures = {
  mockAuthReady: void;
};

async function signInAndLoad(page: Page) {
  await seedOidcSessionViaMockAuth(page);
}

export const test = base.extend<TestFixtures, WorkerFixtures>({
  mockAuthReady: [
    async ({ playwright }, run) => {
      const request = await playwright.request.newContext();
      try {
        await ensureMockAuthEmailStrategy(request);
        await run();
      } finally {
        await request.dispose();
      }
    },
    { scope: 'worker' },
  ],
  page: async ({ page, mockAuthReady: _mockAuthReady }, runPage) => {
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
