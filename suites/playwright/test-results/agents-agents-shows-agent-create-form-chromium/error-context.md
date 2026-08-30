# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: agents.spec.ts >> agents >> shows agent create form
- Location: test/e2e/agents.spec.ts:40:3

# Error details

```
Test timeout of 60000ms exceeded while setting up "page".
```

```
Error: page.waitForURL: Test timeout of 60000ms exceeded.
=========================== logs ===========================
waiting for navigation until "load"
  navigated to "https://auth.agyn.dev:2496/auth/local/login?back=&state=x22hgvxx3x372p3ljuy25axfk"
============================================================
```

# Page snapshot

```yaml
- generic [active] [ref=e1]:
  - img [ref=e4]
  - generic [ref=e6]:
    - heading "Internal Server Error" [level=2] [ref=e7]
    - paragraph [ref=e8]: Login error.
```

# Test source

```ts
  113 |   // admin's.
  114 |   const password = process.env.E2E_OIDC_PASSWORD ?? dexPasswords[address] ?? '';
  115 |   if (!password) {
  116 |     throw new Error(
  117 |       `no password for ${address}: the bundled accounts are ${Object.keys(dexPasswords).join(' and ')}; ` +
  118 |         'set E2E_OIDC_PASSWORD for any other',
  119 |     );
  120 |   }
  121 |   await page.locator(dexLoginField).fill(address);
  122 |   await page.locator(dexPasswordField).fill(password);
  123 |   await page.locator(dexSubmitButton).click();
  124 | }
  125 | 
  126 | async function fillLoginForm(
  127 |   page: Page,
  128 |   expectedEmail: string,
  129 |   onLoginPage?: (page: Page) => Promise<void>,
  130 | ): Promise<void> {
  131 |   if (onLoginPage) {
  132 |     await onLoginPage(page);
  133 |   }
  134 | 
  135 |   if (await isLocatorVisible(page.locator(dexPasswordField), 2000)) {
  136 |     await fillDexLoginForm(page, expectedEmail);
  137 |     return;
  138 |   }
  139 | 
  140 |   const strategyTabs = page.getByTestId('login-strategy-tabs');
  141 |   if (await isLocatorVisible(strategyTabs, 2000)) {
  142 |     const emailTab = strategyTabs.getByRole('tab', { name: 'Email' });
  143 |     if (await isLocatorVisible(emailTab, 2000)) {
  144 |       await emailTab.click();
  145 |     }
  146 |   }
  147 | 
  148 |   const emailInput = page.getByTestId('login-email-input');
  149 |   if ((await emailInput.count()) > 0) {
  150 |     await expect(emailInput).toBeVisible({ timeout: 5000 });
  151 |     await emailInput.fill(expectedEmail);
  152 |   } else {
  153 |     const usernameInput = page.getByTestId('login-username-input');
  154 |     await expect(usernameInput).toBeVisible({ timeout: 5000 });
  155 |     await usernameInput.fill(expectedEmail);
  156 |   }
  157 | 
  158 |   await page.getByRole('button', { name: 'Continue' }).click();
  159 | }
  160 | 
  161 | async function waitForOidcSession(page: Page, timeoutMs: number): Promise<void> {
  162 |   await expect
  163 |     .poll(async () => {
  164 |       const session = await readOidcSession(page);
  165 |       return session?.accessToken ?? '';
  166 |     }, { timeout: timeoutMs })
  167 |     .not.toBe('');
  168 | }
  169 | 
  170 | export async function completeOidcLogin(page: Page, options: BrowserLoginOptions = {}): Promise<boolean> {
  171 |   const expectedEmail = options.email ?? process.env.E2E_OIDC_EMAIL ?? defaultEmail;
  172 |   const timeoutMs = options.timeoutMs ?? 30000;
  173 |   const loginReady = await waitForLoginForm(page, timeoutMs);
  174 |   if (!loginReady) {
  175 |     return false;
  176 |   }
  177 |   await fillLoginForm(page, expectedEmail, options.onLoginPage);
  178 |   return true;
  179 | }
  180 | 
  181 | export async function signInViaOidc(page: Page, email?: string, options: SignInOptions = {}): Promise<boolean> {
  182 |   const expectedEmail = email ?? process.env.E2E_OIDC_EMAIL ?? defaultEmail;
  183 |   const forceLogin = options.force ?? false;
  184 |   // Off by default. It does not pick an account, it promotes whoever signed in
  185 |   // using the bootstrap token, so leaving it on made every spec a cluster admin
  186 |   // and no spec has ever exercised the authorization the services enforce.
  187 |   const ensureAdmin = options.ensureAdmin ?? false;
  188 | 
  189 |   await page.goto('/');
  190 |   if (forceLogin) {
  191 |     await clearAuthState(page);
  192 |     await page.goto('/');
  193 |   }
  194 | 
  195 |   const pageTitle = page.getByTestId('page-title');
  196 |   const sidebarNav = page.getByTestId('console-sidebar');
  197 |   const noAccessState = page.getByTestId('console-no-access');
  198 |   const appReady = pageTitle.or(sidebarNav).or(noAccessState);
  199 | 
  200 |   let initialState: 'app' | 'login' | null = await Promise.race([
  201 |     waitForAppReady(appReady, 10000),
  202 |     waitForLoginForm(page, 10000).then((ready) => (ready ? ('login' as const) : null)),
  203 |   ]);
  204 | 
  205 |   if (initialState !== 'login' && initialState !== 'app') {
  206 |     const loginReady = await waitForLoginForm(page, 15000);
  207 |     if (loginReady) {
  208 |       initialState = 'login';
  209 |     }
  210 |   }
  211 | 
  212 |   if (initialState === 'login') {
> 213 |     const callbackPromise = page.waitForURL(/\/callback/, { timeout: 60000 }).catch((error) => {
      |                                  ^ Error: page.waitForURL: Test timeout of 60000ms exceeded.
  214 |       if (isTimeoutError(error)) {
  215 |         return null;
  216 |       }
  217 |       throw error;
  218 |     });
  219 |     const completed = await completeOidcLogin(page, { email: expectedEmail, onLoginPage: options.onLoginPage });
  220 |     if (completed) {
  221 |       await callbackPromise;
  222 |       await waitForOidcSession(page, 60000);
  223 |     }
  224 |   }
  225 | 
  226 |   await page.goto('/');
  227 |   await expect(appReady.first()).toBeVisible({ timeout: 30000 });
  228 | 
  229 |   if (ensureAdmin) {
  230 |     await ensureClusterAdmin(page);
  231 |   }
  232 | 
  233 |   return initialState === 'login';
  234 | }
  235 | 
  236 | // The bundled cluster admin, for the features that are the admin's. Named
  237 | // rather than assembled at each call site so a spec says which account it needs
  238 | // and nothing else has to know the address or the password.
  239 | export async function signInAsClusterAdmin(page: Page): Promise<boolean> {
  240 |   return signInViaOidc(page, clusterAdminEmail, { ensureAdmin: true });
  241 | }
  242 | 
```