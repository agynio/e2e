import type { Browser, Page } from '@playwright/test';
import { test as base, expect } from '@playwright/test';
import { signInViaOidc } from './sign-in-helper';

// Two people, and the platform ships exactly two accounts. mockauth invents an
// account for any address it is given; Dex knows only the ones the chart
// declares, so a pair of made-up addresses signs nobody in. Overridable for an
// environment with real accounts of its own.
const USER_A_EMAIL = process.env.E2E_USER_A_EMAIL ?? 'user@agyn.dev';
const USER_B_EMAIL = process.env.E2E_USER_B_EMAIL ?? 'admin@agyn.dev';

type MultiUserFixtures = {
  userAPage: Page;
  userBPage: Page;
};

async function createUserContext(browser: Browser, email: string) {
  const context = await browser.newContext({ ignoreHTTPSErrors: true });
  const page = await context.newPage();
  page.on('console', (msg) => {
    if (msg.type() === 'error') console.log('[browser-error]', msg.text());
  });
  page.on('requestfailed', (request) => {
    console.log(`[request-failed] ${request.url()} — ${request.failure()?.errorText}`);
  });
  await signInViaOidc(page, email);
  return { page, context };
}

export const test = base.extend<MultiUserFixtures>({
  userAPage: async ({ browser }, use) => {
    const { page, context } = await createUserContext(browser, USER_A_EMAIL);
    await use(page);
    await context.close();
  },

  userBPage: async ({ browser, userAPage }, use) => {
    void userAPage;
    const { page, context } = await createUserContext(browser, USER_B_EMAIL);
    await use(page);
    await context.close();
  },
});

export { expect, USER_A_EMAIL };
