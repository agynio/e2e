import { argosScreenshot } from '@argos-ci/playwright';
import { test, expect } from '@playwright/test';
import { signInViaOidc } from './sign-in-helper';
import { readOidcSession } from './oidc-helpers';

const defaultEmail = 'e2e-tester@agyn.test';
const expectedEmail = process.env.E2E_OIDC_EMAIL ?? defaultEmail;

test.describe('sign-in', { tag: ['@svc_console', '@smoke'] }, () => {
  test('signs in via oidc redirect flow', async ({ page }) => {
    test.setTimeout(60_000);
    await signInViaOidc(page, expectedEmail, {
      onLoginPage: async (loginPage) => {
        const loginHeading = loginPage.getByRole('heading', { level: 1 });
        await expect(loginHeading).toContainText('Log in to');
      },
    });
    await argosScreenshot(page, 'sign-in-complete');

    const storedUser = await readOidcSession(page);

    expect(storedUser).not.toBeNull();
    expect(storedUser?.accessToken).toBeTruthy();
  });
});
