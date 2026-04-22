import { argosScreenshot } from '@argos-ci/playwright';
import { test, expect } from '@playwright/test';
import { signInViaMockAuth } from './sign-in-helper';
import { readOidcSession } from './oidc-helpers';

const defaultEmail = 'e2e-tester@agyn.test';
const expectedEmail = process.env.E2E_OIDC_EMAIL ?? defaultEmail;

test.describe('sign-in', { tag: ['@svc_console', '@smoke'] }, () => {
  test('signs in via mockauth redirect flow', async ({ page }) => {
    test.setTimeout(60_000);
    const signedIn = await signInViaMockAuth(page, expectedEmail, {
      onLoginPage: async (loginPage) => {
        const loginHeading = loginPage.getByRole('heading', { level: 1 });
        await expect(loginHeading).toContainText('Log in to');
      },
    });
    await argosScreenshot(page, 'sign-in-complete');

    const storedUser = await readOidcSession(page);

    if (!signedIn) {
      expect(storedUser).toBeNull();
      return;
    }

    expect(storedUser).not.toBeNull();
    expect(storedUser?.accessToken).toBeTruthy();
  });
});
