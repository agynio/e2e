import type { ConsoleMessage, Page } from '@playwright/test';
import { test, expect } from './fixtures';
import { createOrganization, setSelectedOrganization } from './console-api';

const TOOLTIP_PROVIDER_ERROR = '`Tooltip` must be used within `TooltipProvider`';
const APP_READY_TIMEOUT_MS = 15000;

type NavigationTarget = {
  navTestId: string;
  title: string;
  pathPattern: RegExp;
};

const platformTargets: NavigationTarget[] = [
  { navTestId: 'nav-dashboard', title: 'Dashboard', pathPattern: /\/$/ },
  { navTestId: 'nav-users', title: 'Users', pathPattern: /\/users$/ },
  { navTestId: 'nav-organizations', title: 'Organizations', pathPattern: /\/organizations$/ },
  { navTestId: 'nav-cluster-runners', title: 'Cluster Runners', pathPattern: /\/runners$/ },
  { navTestId: 'nav-apps', title: 'Apps', pathPattern: /\/apps$/ },
];

const organizationTargets: NavigationTarget[] = [
  { navTestId: 'nav-organization-overview', title: 'Overview', pathPattern: /\/organizations\/[^/]+$/ },
  { navTestId: 'nav-organization-members', title: 'Members', pathPattern: /\/organizations\/[^/]+\/members$/ },
  { navTestId: 'nav-organization-agents', title: 'Agents', pathPattern: /\/organizations\/[^/]+\/agents$/ },
  { navTestId: 'nav-organization-volumes', title: 'Volumes', pathPattern: /\/organizations\/[^/]+\/volumes$/ },
  { navTestId: 'nav-organization-runners', title: 'Runners', pathPattern: /\/organizations\/[^/]+\/runners$/ },
  { navTestId: 'nav-organization-apps', title: 'Apps', pathPattern: /\/organizations\/[^/]+\/apps$/ },
  { navTestId: 'nav-organization-llm-providers', title: 'LLM Providers', pathPattern: /\/organizations\/[^/]+\/llm-providers$/ },
  { navTestId: 'nav-organization-models', title: 'Models', pathPattern: /\/organizations\/[^/]+\/models$/ },
  { navTestId: 'nav-organization-secret-providers', title: 'Secret Providers', pathPattern: /\/organizations\/[^/]+\/secret-providers$/ },
  { navTestId: 'nav-organization-secrets', title: 'Secrets', pathPattern: /\/organizations\/[^/]+\/secrets$/ },
  { navTestId: 'nav-organization-image-pull-secrets', title: 'Image Pull Secrets', pathPattern: /\/organizations\/[^/]+\/image-pull-secrets$/ },
  { navTestId: 'nav-organization-workloads', title: 'Workloads', pathPattern: /\/organizations\/[^/]+\/activity\/workloads$/ },
  { navTestId: 'nav-organization-storage', title: 'Storage', pathPattern: /\/organizations\/[^/]+\/activity\/storage$/ },
  { navTestId: 'nav-organization-threads', title: 'Threads', pathPattern: /\/organizations\/[^/]+\/threads$/ },
  { navTestId: 'nav-organization-usage', title: 'Usage', pathPattern: /\/organizations\/[^/]+\/usage$/ },
];

function formatConsoleMessage(message: ConsoleMessage): string {
  return `[${message.type()}] ${message.text()}`;
}

function collectCrashSignals(page: Page): string[] {
  const crashSignals: string[] = [];

  page.on('pageerror', (error) => {
    crashSignals.push(error.message);
  });
  page.on('console', (message) => {
    const text = message.text();
    if (text.includes(TOOLTIP_PROVIDER_ERROR)) {
      crashSignals.push(formatConsoleMessage(message));
    }
  });
  page.on('crash', () => {
    crashSignals.push('Page crashed.');
  });

  return crashSignals;
}

async function expectNoCrashSignals(crashSignals: string[], target: NavigationTarget): Promise<void> {
  expect(crashSignals, `unexpected browser crash signals after opening ${target.title}`).toEqual([]);
}

async function expectShellReady(page: Page, target: NavigationTarget): Promise<void> {
  await expect(page).toHaveURL(target.pathPattern, { timeout: APP_READY_TIMEOUT_MS });
  await expect(page.getByTestId('console-sidebar')).toBeVisible({ timeout: APP_READY_TIMEOUT_MS });
  await expect(page.getByTestId('page-title')).toHaveText(target.title, { timeout: APP_READY_TIMEOUT_MS });
}

async function openNavigationTarget(page: Page, target: NavigationTarget, crashSignals: string[]): Promise<void> {
  crashSignals.length = 0;
  await page.getByTestId(target.navTestId).click();
  await expectShellReady(page, target);
  await expectNoCrashSignals(crashSignals, target);
}

test.describe('console navigation', { tag: ['@svc_console', '@svc_gateway', '@smoke'] }, () => {
  test('opens every platform sidebar section without browser crashes', async ({ page }) => {
    const crashSignals = collectCrashSignals(page);

    await page.goto('/');
    await expectShellReady(page, platformTargets[0]);
    await expectNoCrashSignals(crashSignals, platformTargets[0]);

    for (const target of platformTargets) {
      await openNavigationTarget(page, target, crashSignals);
    }
  });

  test('opens every organization sidebar section without browser crashes', async ({ page }) => {
    const crashSignals = collectCrashSignals(page);
    const organizationId = await createOrganization(page, `e2e-org-navigation-${Date.now()}`);
    await setSelectedOrganization(page, organizationId);

    await page.goto(`/organizations/${organizationId}`);
    await expectShellReady(page, organizationTargets[0]);
    await expectNoCrashSignals(crashSignals, organizationTargets[0]);

    for (const target of organizationTargets) {
      await openNavigationTarget(page, target, crashSignals);
    }
  });
});
