import type { Page } from '@playwright/test';
import { test as base, expect } from '@playwright/test';
import { signInAsClusterAdmin, signInViaOidc } from './sign-in-helper';

export { expect };

// adminPage is the same page signed in as the bundled cluster admin, for specs
// covering a feature that is the admin's. Everything else takes `page`, which
// is an ordinary member: the services are where authorization is enforced, and
// a suite that is cluster admin throughout cannot tell a rule that holds from
// one that was never asked.
type TestFixtures = {
  adminPage: Page;
};

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
  adminPage: async ({ browser }, runAdminPage) => {
    const context = await browser.newContext();
    const adminPage = await context.newPage();
    try {
      await signInAsClusterAdmin(adminPage);
      await runAdminPage(adminPage);
    } finally {
      await context.close();
    }
  },
});
