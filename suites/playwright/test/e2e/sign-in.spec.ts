import { argosScreenshot } from '@argos-ci/playwright';
import { test, expect } from '@playwright/test';
import { signInViaOidc } from './sign-in-helper';
import { readOidcSession } from './oidc-helpers';

// Named only when the environment names one. Left unset the helper signs in as
// the account the platform ships, which is the only one Dex will accept.
const expectedEmail = process.env.E2E_OIDC_EMAIL;

test.describe('sign-in', { tag: ['@svc_console', '@smoke'] }, () => {
  test('signs in via oidc redirect flow', async ({ page }) => {
    test.setTimeout(60_000);
    await signInViaOidc(page, expectedEmail, {
      onLoginPage: async (loginPage) => {
        // Any level: mockauth's is an h1, Dex's an h2.
        const loginHeading = loginPage.getByRole('heading', { name: /Log in to/i });
        await expect(loginHeading.first()).toBeVisible();
      },
    });
    await argosScreenshot(page, 'sign-in-complete');

    const storedUser = await readOidcSession(page);

    expect(storedUser).not.toBeNull();
    expect(storedUser?.accessToken).toBeTruthy();
  });
});
