import type { APIRequestContext, Locator, Page } from '@playwright/test';
import { expect } from '@playwright/test';
import { User, type IdTokenClaims } from 'oidc-client-ts';
import { createHash, randomBytes } from 'node:crypto';
import { readOidcSession } from './oidc-helpers';

const defaultEmail = 'e2e-tester@agyn.test';

type SeedOidcOptions = {
  onLoginPage?: (page: Page) => Promise<void>;
  force?: boolean;
  email?: string;
  landingPath?: string;
};

type BrowserLoginOptions = {
  onLoginPage?: (page: Page) => Promise<void>;
  email?: string;
};

type OidcRuntimeConfig = {
  authority: string;
  clientId: string;
  scope: string;
};

type TokenResponse = {
  access_token?: string;
  id_token?: string;
  refresh_token?: string;
  token_type?: string;
  scope?: string;
  expires_in?: number;
  session_state?: string;
};

function resolveBaseUrl(): string {
  const baseUrl = process.env.E2E_BASE_URL;
  if (!baseUrl) {
    throw new Error('E2E_BASE_URL is required to run e2e tests.');
  }
  return baseUrl;
}

function stripTrailingSlash(value: string): string {
  return value.replace(/\/+$/, '');
}

function base64UrlEncode(buffer: Buffer): string {
  return buffer
    .toString('base64')
    .replace(/\+/g, '-')
    .replace(/\//g, '_')
    .replace(/=+$/g, '');
}

function createPkcePair(): { codeVerifier: string; codeChallenge: string } {
  const codeVerifier = base64UrlEncode(randomBytes(32));
  const codeChallenge = base64UrlEncode(createHash('sha256').update(codeVerifier).digest());
  return { codeVerifier, codeChallenge };
}

function randomState(length = 16): string {
  return base64UrlEncode(randomBytes(length));
}

function decodeJwtPayload(token: string): IdTokenClaims {
  const parts = token.split('.');
  if (parts.length < 2) {
    throw new Error('MockAuth id token is malformed.');
  }
  const payload = parts[1];
  const normalized = payload.replace(/-/g, '+').replace(/_/g, '/');
  const padded = normalized.padEnd(normalized.length + ((4 - (normalized.length % 4)) % 4), '=');
  const decoded = Buffer.from(padded, 'base64').toString('utf8');
  const parsed = JSON.parse(decoded);
  if (!parsed || typeof parsed !== 'object') {
    throw new Error('MockAuth id token payload is invalid.');
  }
  const claims = parsed as Record<string, unknown>;
  const required: Array<keyof IdTokenClaims> = ['sub', 'iss', 'aud', 'exp', 'iat'];
  for (const key of required) {
    if (typeof claims[key] === 'undefined') {
      throw new Error(`MockAuth id token missing ${key}.`);
    }
  }
  return claims as IdTokenClaims;
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

async function fillMockAuthLoginForm(
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

function readEnvValue(body: string, key: string): string | undefined {
  const matcher = new RegExp(`${key}:\\s*"([^"]*)"`);
  const match = body.match(matcher);
  return match ? match[1] : undefined;
}

function isTimeoutError(error: unknown): error is Error {
  return error instanceof Error && error.name === 'TimeoutError';
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

async function resolveRuntimeEnv(request: APIRequestContext): Promise<Record<string, string | undefined>> {
  const response = await request.get(new URL('/env.js', resolveBaseUrl()).toString());
  if (!response.ok()) {
    throw new Error(`Failed to load runtime env.js (${response.status()}).`);
  }
  const body = await response.text();
  return {
    OIDC_AUTHORITY: readEnvValue(body, 'OIDC_AUTHORITY'),
    OIDC_CLIENT_ID: readEnvValue(body, 'OIDC_CLIENT_ID'),
    OIDC_SCOPE: readEnvValue(body, 'OIDC_SCOPE'),
  };
}

async function resolveOidcConfig(request: APIRequestContext): Promise<OidcRuntimeConfig> {
  const env = await resolveRuntimeEnv(request);
  const authority = stripTrailingSlash(process.env.E2E_OIDC_AUTHORITY ?? env.OIDC_AUTHORITY ?? '');
  const clientId = process.env.E2E_OIDC_CLIENT_ID ?? env.OIDC_CLIENT_ID ?? '';
  const scope = process.env.E2E_OIDC_SCOPE ?? env.OIDC_SCOPE ?? '';

  if (!authority || !clientId || !scope) {
    throw new Error('OIDC config is missing (authority, client ID, or scope).');
  }
  return { authority, clientId, scope };
}

export async function ensureMockAuthEmailStrategy(request: APIRequestContext): Promise<void> {
  const config = await resolveOidcConfig(request);
  const mockAuthOrigin = new URL(config.authority).origin;
  const response = await request.post(new URL('/api/test/client-auth-strategies', mockAuthOrigin).toString(), {
    headers: { 'Content-Type': 'application/json' },
    data: {
      clientId: config.clientId,
      strategies: {
        username: { enabled: true, subSource: 'entered' },
        email: { enabled: true, subSource: 'entered', emailVerifiedMode: 'true' },
      },
    },
  });
  if (response.status() === 404) {
    const body = await response.text();
    console.warn(`MockAuth test routes disabled; skipping email strategy enablement. (${body})`);
    return;
  }
  if (!response.ok()) {
    const body = await response.text();
    throw new Error(`Failed to enable MockAuth email strategy (${response.status()}): ${body}`);
  }
}

function resolveRedirectUri(): string {
  return new URL('/callback', resolveBaseUrl()).toString();
}

async function waitForRedirectResponse(page: Page, redirectUri: string) {
  return page.waitForResponse((response) => {
    if (response.status() < 300 || response.status() >= 400) return false;
    const location = response.headers()['location'];
    return Boolean(location && location.startsWith(redirectUri));
  });
}

async function exchangeAuthCode(
  config: OidcRuntimeConfig,
  params: { code: string; codeVerifier: string; redirectUri: string },
): Promise<TokenResponse> {
  const tokenUrl = `${config.authority}/token`;
  const body = new URLSearchParams({
    grant_type: 'authorization_code',
    client_id: config.clientId,
    redirect_uri: params.redirectUri,
    code: params.code,
    code_verifier: params.codeVerifier,
  });
  const response = await fetch(tokenUrl, {
    method: 'POST',
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
    body,
  });
  if (!response.ok) {
    const text = await response.text();
    throw new Error(`MockAuth token exchange failed (${response.status}): ${text}`);
  }
  return (await response.json()) as TokenResponse;
}

function buildUserStorage(config: OidcRuntimeConfig, tokens: TokenResponse): { storageKey: string; storageValue: string } {
  if (!tokens.access_token || !tokens.id_token) {
    throw new Error('MockAuth token response missing access or id token.');
  }
  const profile = decodeJwtPayload(tokens.id_token);
  const expiresAt = tokens.expires_in ? Math.floor(Date.now() / 1000) + tokens.expires_in : undefined;
  const user = new User({
    id_token: tokens.id_token,
    access_token: tokens.access_token,
    refresh_token: tokens.refresh_token,
    token_type: tokens.token_type ?? 'Bearer',
    scope: tokens.scope,
    profile,
    expires_at: expiresAt,
    session_state: tokens.session_state ?? null,
  });
  return {
    storageKey: `oidc.user:${config.authority}:${config.clientId}`,
    storageValue: user.toStorageString(),
  };
}

async function seedOidcSession(
  page: Page,
  tokens: TokenResponse,
  config: OidcRuntimeConfig,
  options: SeedOidcOptions,
): Promise<void> {
  const { storageKey, storageValue } = buildUserStorage(config, tokens);
  const expectedOrigin = new URL(resolveBaseUrl()).origin;

  await page.addInitScript(
    ({ key, value, origin }) => {
      if (window.location.origin !== origin) return;
      const seededKey = 'e2e:oidc-seeded';
      if (window.sessionStorage.getItem(seededKey)) return;
      window.sessionStorage.setItem(key, value);
      window.sessionStorage.setItem(seededKey, 'true');
    },
    { key: storageKey, value: storageValue, origin: expectedOrigin },
  );

  const landingPath = options.landingPath ?? '/';
  await page.goto(landingPath);
  const session = await readOidcSession(page);
  if (!session?.accessToken) {
    throw new Error('MockAuth session storage was not initialized.');
  }
}

export async function seedOidcSessionViaMockAuth(page: Page, options: SeedOidcOptions = {}): Promise<void> {
  const expectedEmail = options.email ?? process.env.E2E_OIDC_EMAIL ?? defaultEmail;
  const config = await resolveOidcConfig(page.context().request);
  const redirectUri = resolveRedirectUri();
  const { codeVerifier, codeChallenge } = createPkcePair();
  const state = randomState();
  const nonce = randomState();

  if (options.force) {
    await clearAuthState(page);
  }

  const authorizeUrl = new URL(`${config.authority}/authorize`);
  authorizeUrl.searchParams.set('client_id', config.clientId);
  authorizeUrl.searchParams.set('redirect_uri', redirectUri);
  authorizeUrl.searchParams.set('response_type', 'code');
  authorizeUrl.searchParams.set('scope', config.scope);
  authorizeUrl.searchParams.set('state', state);
  authorizeUrl.searchParams.set('nonce', nonce);
  authorizeUrl.searchParams.set('code_challenge', codeChallenge);
  authorizeUrl.searchParams.set('code_challenge_method', 'S256');

  const redirectResponsePromise = waitForRedirectResponse(page, redirectUri);
  await page.goto(authorizeUrl.toString());

  const loginHeading = page.getByRole('heading', { level: 1, name: /Log in to/ });
  const loginReady = await Promise.race([
    waitForLocator(loginHeading, 10000),
    redirectResponsePromise.then(() => false),
  ]);

  if (loginReady) {
    await fillMockAuthLoginForm(page, expectedEmail, options.onLoginPage);
  }

  const redirectResponse = await redirectResponsePromise;
  const location = redirectResponse.headers()['location'];
  if (!location) {
    throw new Error('MockAuth redirect missing location header.');
  }
  const callback = new URL(location);
  const code = callback.searchParams.get('code');
  const returnedState = callback.searchParams.get('state');
  if (!code || !returnedState) {
    throw new Error('MockAuth callback missing code or state.');
  }
  if (returnedState !== state) {
    throw new Error('MockAuth callback state mismatch.');
  }

  const tokens = await exchangeAuthCode(config, { code, codeVerifier, redirectUri });
  await seedOidcSession(page, tokens, config, options);
}

export async function completeMockAuthLogin(page: Page, options: BrowserLoginOptions = {}): Promise<void> {
  const expectedEmail = options.email ?? process.env.E2E_OIDC_EMAIL ?? defaultEmail;
  const loginHeading = page.getByRole('heading', { level: 1, name: /Log in to/ });
  await expect(loginHeading).toBeVisible({ timeout: 30000 });
  await fillMockAuthLoginForm(page, expectedEmail, options.onLoginPage);
}
