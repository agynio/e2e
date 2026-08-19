import type { Locator, Page } from '@playwright/test';
import { expect } from '@playwright/test';

const defaultEmail = 'e2e-tester@agyn.test';

// Dex's own password form, which the bundle VM serves. Not the mockauth form
// below: an h2 rather than an h1, a password field, and no testids to hang a
// locator on -- so it is matched on the ids Dex's template gives them.
const dexPasswordField = 'input#password';
const dexLoginField = 'input#login';
const dexSubmitButton = 'button#submit-login';

// The accounts the platform charts ship. The ordinary member by default: a
// suite signed in as cluster admin cannot tell a rule that holds from one that
// was never asked.
const dexPasswords: Record<string, string> = {
  'user@agyn.dev': 'user',
  'admin@agyn.dev': 'admin',
};
const defaultDexEmail = 'user@agyn.dev';

async function fillDexLoginForm(page: Page, email: string): Promise<void> {
  const address = email === defaultEmail ? defaultDexEmail : email;
  const password = process.env.E2E_OIDC_PASSWORD ?? dexPasswords[address] ?? '';
  if (!password) {
    throw new Error(
      `no password for ${address}: the bundled accounts are ${Object.keys(dexPasswords).join(' and ')}; ` +
        'set E2E_OIDC_PASSWORD for any other',
    );
  }
  await page.locator(dexLoginField).fill(address);
  await page.locator(dexPasswordField).fill(password);
  await page.locator(dexSubmitButton).click();
}


type SignInOptions = {
  onLoginPage?: (page: Page) => Promise<void>;
  force?: boolean;
};

type BrowserLoginOptions = {
  onLoginPage?: (page: Page) => Promise<void>;
  email?: string;
  timeoutMs?: number;
};

function isTimeoutError(error: unknown): error is Error {
  return error instanceof Error && error.name === 'TimeoutError';
}

async function waitForLocator(locator: Locator, timeout: number): Promise<boolean> {
  try {
    await locator.waitFor({ timeout });
    return true;
  } catch (error) {
    if (isTimeoutError(error)) {
      return false;
    }
    throw error;
  }
}

async function isLocatorVisible(locator: Locator, timeout: number): Promise<boolean> {
  try {
    return await locator.isVisible({ timeout });
  } catch (error) {
    if (isTimeoutError(error)) {
      return false;
    }
    throw error;
  }
}

async function clearAuthState(page: Page): Promise<void> {
  await page.evaluate(() => {
    window.sessionStorage.clear();
    window.localStorage.clear();
  });
  await page.context().clearCookies();
}

async function waitForLoginForm(page: Page, timeoutMs: number): Promise<boolean> {
  const loginHeading = page.getByRole('heading', { level: 1, name: /Log in to/i });
  const emailInput = page.getByTestId('login-email-input');
  const usernameInput = page.getByTestId('login-username-input');
  return Promise.race([
    waitForLocator(loginHeading, timeoutMs),
    waitForLocator(emailInput, timeoutMs),
    waitForLocator(usernameInput, timeoutMs),
    waitForLocator(page.locator(dexPasswordField), timeoutMs),
  ]);
}

async function fillLoginForm(
  page: Page,
  expectedEmail: string,
  onLoginPage?: (page: Page) => Promise<void>,
): Promise<void> {
  if (onLoginPage) {
    await onLoginPage(page);
  }

  if (await isLocatorVisible(page.locator(dexPasswordField), 2000)) {
    await fillDexLoginForm(page, expectedEmail);
    return;
  }

  const strategyTabs = page.getByTestId('login-strategy-tabs');
  if (await isLocatorVisible(strategyTabs, 2000)) {
    const emailTab = strategyTabs.getByRole('tab', { name: 'Email' });
    if (await isLocatorVisible(emailTab, 2000)) {
      await emailTab.click();
    }
  }

  const emailInput = page.getByTestId('login-email-input');
  if ((await emailInput.count()) > 0) {
    await expect(emailInput).toBeVisible({ timeout: 5000 });
    await emailInput.fill(expectedEmail);
  } else {
    const usernameInput = page.getByTestId('login-username-input');
    await expect(usernameInput).toBeVisible({ timeout: 5000 });
    await usernameInput.fill(expectedEmail);
  }

  await page.getByRole('button', { name: 'Continue' }).click();
}

export async function completeOidcLogin(page: Page, options: BrowserLoginOptions = {}): Promise<boolean> {
  const expectedEmail = options.email ?? process.env.E2E_OIDC_EMAIL ?? defaultEmail;
  const timeoutMs = options.timeoutMs ?? 30000;
  const loginReady = await waitForLoginForm(page, timeoutMs);
  if (!loginReady) {
    return false;
  }
  await fillLoginForm(page, expectedEmail, options.onLoginPage);
  return true;
}

export async function signInViaOidc(page: Page, email?: string, options: SignInOptions = {}): Promise<boolean> {
  const expectedEmail = email ?? process.env.E2E_OIDC_EMAIL ?? defaultEmail;
  const forceLogin = options.force ?? false;

  await page.goto('/');
  if (forceLogin) {
    await clearAuthState(page);
    await page.goto('/');
  }

  const chatList = page.getByTestId('chat-list');
  const noOrganizationsScreen = page.getByTestId('no-organizations-screen');
  const appReady = chatList.or(noOrganizationsScreen);

  let initialState: 'app' | 'login' | null = await Promise.race([
    appReady
      .waitFor({ timeout: 10000 })
      .then(() => 'app' as const)
      .catch((error) => {
        if (isTimeoutError(error)) {
          return null;
        }
        throw error;
      }),
    waitForLoginForm(page, 10000).then((ready) => (ready ? ('login' as const) : null)),
  ]);

  if (initialState === 'app' && !forceLogin) {
    await expect(appReady).toBeVisible({ timeout: 30000 });
    return false;
  }

  if (initialState !== 'login') {
    const loginReady = await waitForLoginForm(page, 15000);
    if (!loginReady) {
      await expect(appReady).toBeVisible({ timeout: 30000 });
      return false;
    }
    initialState = 'login';
  }

  const callbackPromise = page.waitForURL(/\/callback/, { timeout: 60000 }).catch((error) => {
    if (isTimeoutError(error)) {
      return null;
    }
    throw error;
  });
  const completed = await completeOidcLogin(page, { email: expectedEmail, onLoginPage: options.onLoginPage });
  if (completed) {
    await callbackPromise;
  }

  await page.waitForURL(/\/chats/, { timeout: 60000 });
  await expect(appReady).toBeVisible({ timeout: 30000 });
  return true;
}
