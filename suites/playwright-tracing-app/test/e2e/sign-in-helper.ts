import type { Locator, Page } from '@playwright/test';
import { expect } from '@playwright/test';
import { readOidcSession } from './oidc-helpers';

const defaultEmail = 'e2e-tester@agyn.test';

type SignInOptions = {
  onLoginPage?: (page: Page) => Promise<void>;
  force?: boolean;
  email?: string;
  landingPath?: string;
};

type BrowserLoginOptions = {
  onLoginPage?: (page: Page) => Promise<void>;
  email?: string;
  timeoutMs?: number;
};

function resolveBaseUrl(): string {
  const baseUrl = process.env.E2E_BASE_URL;
  if (!baseUrl) {
    throw new Error('E2E_BASE_URL is required to run e2e tests.');
  }
  return baseUrl;
}

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

export async function clearAuthState(page: Page): Promise<void> {
  await page.context().clearCookies();
  const expectedOrigin = new URL(resolveBaseUrl()).origin;
  const clearedKey = 'e2e:oidc-cleared';
  const clearStorage = ({ origin, key }: { origin: string; key: string }) => {
    if (window.location.origin !== origin) return;
    if (window.sessionStorage.getItem(key)) return;
    window.sessionStorage.clear();
    window.localStorage.clear();
    window.sessionStorage.setItem(key, 'true');
  };

  await page.addInitScript(clearStorage, { origin: expectedOrigin, key: clearedKey });

  const currentOrigin = new URL(page.url()).origin;
  if (currentOrigin === expectedOrigin) {
    await page.evaluate(clearStorage, { origin: expectedOrigin, key: clearedKey });
  }
}

async function waitForLoginForm(page: Page, timeoutMs: number): Promise<boolean> {
  const loginHeading = page.getByRole('heading', { level: 1, name: /Log in to/i });
  const emailInput = page.getByTestId('login-email-input');
  const usernameInput = page.getByTestId('login-username-input');
  return Promise.race([
    waitForLocator(loginHeading, timeoutMs),
    waitForLocator(emailInput, timeoutMs),
    waitForLocator(usernameInput, timeoutMs),
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

async function waitForOidcSession(page: Page, timeoutMs: number): Promise<void> {
  await expect
    .poll(async () => {
      const session = await readOidcSession(page);
      return session?.accessToken ?? '';
    }, { timeout: timeoutMs })
    .not.toBe('');
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

export async function signInViaOidc(page: Page, options: SignInOptions = {}): Promise<boolean> {
  const expectedEmail = options.email ?? process.env.E2E_OIDC_EMAIL ?? defaultEmail;
  const forceLogin = options.force ?? false;
  const landingPath = options.landingPath ?? '/';

  await page.goto(landingPath);
  if (forceLogin) {
    await clearAuthState(page);
    await page.goto(landingPath);
  }

  let loginReady = await waitForLoginForm(page, 10000);
  if (!loginReady) {
    loginReady = await waitForLoginForm(page, 15000);
  }

  if (loginReady) {
    const callbackPromise = page.waitForURL(/\/callback/, { timeout: 60000 }).catch(() => null);
    const completed = await completeOidcLogin(page, { email: expectedEmail, onLoginPage: options.onLoginPage });
    if (completed) {
      await callbackPromise;
      await waitForOidcSession(page, 60000);
    }
    return true;
  }

  await waitForOidcSession(page, 30000);
  return false;
}
