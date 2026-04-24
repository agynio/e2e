import { createArgosReporterOptions } from '@argos-ci/playwright/reporter';
import { defineConfig, devices } from '@playwright/test';

const baseUrl = process.env.E2E_BASE_URL;
if (!baseUrl) {
  throw new Error('E2E_BASE_URL is required to run Playwright e2e tests.');
}

export default defineConfig({
  testDir: './test/e2e',
  timeout: 60000,
  fullyParallel: true,
  forbidOnly: Boolean(process.env.CI),
  retries: process.env.CI ? 1 : 0,
  workers: process.env.CI ? 2 : undefined,
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
    baseURL: baseUrl,
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
    ignoreHTTPSErrors: true,
  },

  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],
});
