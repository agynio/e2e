import type { Locator, Page } from '@playwright/test';
import { expect } from '@playwright/test';
import { ensureClusterAdmin } from './console-api';
import { readOidcSession } from './oidc-helpers';

const defaultEmail = 'e2e-tester@agyn.test';

// Dex's own password form, which the bundle VM serves. It is not the mockauth
// form the rest of this helper drives: an h2 rather than an h1, a password
// field, and no testids to hang a locator on -- so it is matched on the ids
// Dex's template gives them.
const dexPasswordField = 'input#password';
const dexLoginField = 'input#login';
const dexSubmitButton = 'button#submit-login';

// The address and password the platform charts ship for the bundled admin, and
// what an install nobody has changed still has. A different environment names
// its own.
const defaultDexEmail = 'admin@agyn.dev';
const defaultDexPassword = 'admin';

type SignInOptions = {
  onLoginPage?: (page: Page) => Promise<void>;
  force?: boolean;
  ensureAdmin?: boolean;
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

async function waitForAppReady(appReady: Locator, timeoutMs: number): Promise<'app' | null> {
  try {
    await appReady.waitFor({ timeout: timeoutMs });
    return 'app';
  } catch (error) {
    if (isTimeoutError(error)) {
      return null;
    }
    throw error;
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
    waitForLocator(page.locator(dexPasswordField), timeoutMs),
  ]);
}

// Dex asks for a password; mockauth takes an address and continues. Told apart
// by the password field rather than by the heading, which is the difference
// that decides what has to be typed.
async function fillDexLoginForm(page: Page, email: string): Promise<void> {
  // mockauth accepts any address and invents the account behind it. Dex only
  // knows the accounts the chart declares, so this suite's generic default is
  // not one it can sign in as -- an address nobody asked for becomes the
  // bundled admin, which is the account provisioning.clusterAdmins names.
  const address = email === defaultEmail ? defaultDexEmail : email;
  const password = process.env.E2E_OIDC_PASSWORD ?? defaultDexPassword;
  await page.locator(dexLoginField).fill(address);
  await page.locator(dexPasswordField).fill(password);
  await page.locator(dexSubmitButton).click();
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

export async function signInViaOidc(page: Page, email?: string, options: SignInOptions = {}): Promise<boolean> {
  const expectedEmail = email ?? process.env.E2E_OIDC_EMAIL ?? defaultEmail;
  const forceLogin = options.force ?? false;
  const ensureAdmin = options.ensureAdmin ?? true;

  await page.goto('/');
  if (forceLogin) {
    await clearAuthState(page);
    await page.goto('/');
  }

  const pageTitle = page.getByTestId('page-title');
  const sidebarNav = page.getByTestId('console-sidebar');
  const noAccessState = page.getByTestId('console-no-access');
  const appReady = pageTitle.or(sidebarNav).or(noAccessState);

  let initialState: 'app' | 'login' | null = await Promise.race([
    waitForAppReady(appReady, 10000),
    waitForLoginForm(page, 10000).then((ready) => (ready ? ('login' as const) : null)),
  ]);

  if (initialState !== 'login' && initialState !== 'app') {
    const loginReady = await waitForLoginForm(page, 15000);
    if (loginReady) {
      initialState = 'login';
    }
  }

  if (initialState === 'login') {
    const callbackPromise = page.waitForURL(/\/callback/, { timeout: 60000 }).catch((error) => {
      if (isTimeoutError(error)) {
        return null;
      }
      throw error;
    });
    const completed = await completeOidcLogin(page, { email: expectedEmail, onLoginPage: options.onLoginPage });
    if (completed) {
      await callbackPromise;
      await waitForOidcSession(page, 60000);
    }
  }

  await page.goto('/');
  await expect(appReady.first()).toBeVisible({ timeout: 30000 });

  if (ensureAdmin) {
    await ensureClusterAdmin(page);
  }

  return initialState === 'login';
}
