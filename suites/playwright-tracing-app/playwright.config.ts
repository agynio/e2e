import { createArgosReporterOptions } from '@argos-ci/playwright/reporter';
import { defineConfig, devices } from '@playwright/test';

const BASE_URL = process.env.E2E_BASE_URL;

if (!BASE_URL) {
  throw new Error(
    'E2E_BASE_URL is required. Run tests via: devspace run test-e2e\n' +
      'Or set E2E_BASE_URL manually to the app URL (e.g., https://tracing.agyn.dev).',
  );
}

export default defineConfig({
  testDir: './test/e2e',
  timeout: 60000,
  // Each test starts a real agent workload whose MCP sidecars install their
  // servers at boot. Two of those at once, beside the platform itself, is what
  // the VM ran out of memory for -- the run died with the pod OOMKilled.
  fullyParallel: false,
  forbidOnly: Boolean(process.env.CI),
  retries: 1,
  workers: 1,
  reporter: [
    process.env.CI ? ['dot'] : ['list'],
    ['junit', { outputFile: 'junit.xml' }],
    ['html', { open: 'never' }],
    [
      '@argos-ci/playwright/reporter',
      createArgosReporterOptions({
        uploadToArgos: Boolean(process.env.CI && process.env.ARGOS_TOKEN),
      }),
    ],
  ],
  use: {
    baseURL: BASE_URL,
    ignoreHTTPSErrors: true,
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
  },
  projects: [{ name: 'chromium', use: { ...devices['Desktop Chrome'] } }],
});
