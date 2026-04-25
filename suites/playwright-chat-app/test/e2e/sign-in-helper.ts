import type { Page } from '@playwright/test';
import { expect } from '@playwright/test';

const defaultEmail = 'e2e-tester@agyn.test';

type SignInOptions = {
  onLoginPage?: (page: Page) => Promise<void>;
  force?: boolean;
};

async function clearAuthState(page: Page): Promise<void> {
  await page.evaluate(() => {
    window.sessionStorage.clear();
    window.localStorage.clear();
  });
  await page.context().clearCookies();
}

export async function signInViaMockAuth(
  page: Page,
  email?: string,
  options: SignInOptions = {},
): Promise<boolean> {
  const expectedEmail = email ?? process.env.E2E_OIDC_EMAIL ?? defaultEmail;
  const forceLogin = options.force ?? false;

  await page.goto('/');
  if (forceLogin) {
    await clearAuthState(page);
    await page.goto('/');
  }

  const loginUrlPattern = /mockauth\.dev\/r\/.*\/oidc/;
  const chatList = page.getByTestId('chat-list');
  const noOrganizationsScreen = page.getByTestId('no-organizations-screen');
  const appReady = chatList.or(noOrganizationsScreen);

  const initialRoute = await Promise.race([
    page
      .waitForURL(loginUrlPattern, { timeout: 10000 })
      .then(() => 'login')
      .catch(() => null),
    appReady
      .waitFor({ timeout: 10000 })
      .then(() => 'app')
      .catch(() => null),
  ]);

  if (initialRoute === 'app' && !forceLogin) {
    await expect(appReady).toBeVisible({ timeout: 30000 });
    return false;
  }

  if (!initialRoute || (initialRoute === 'app' && forceLogin)) {
    const loginReached = await page
      .waitForURL(loginUrlPattern, { timeout: 15000 })
      .then(() => true)
      .catch(() => false);
    if (!loginReached) {
      await expect(appReady).toBeVisible({ timeout: 30000 });
      return false;
    }
  }

  if (options.onLoginPage) {
    await options.onLoginPage(page);
  }

  const strategyTabs = page.getByTestId('login-strategy-tabs');
  if (await strategyTabs.isVisible()) {
    await strategyTabs.getByRole('tab', { name: 'Email' }).click();
  }

  const emailInput = page.getByTestId('login-email-input');
  await expect(emailInput).toBeVisible();
  await emailInput.fill(expectedEmail);

  await page.getByRole('button', { name: 'Continue' }).click();

  await page.waitForURL(/\/chats/);
  await expect(appReady).toBeVisible({ timeout: 30000 });
  return true;
}
