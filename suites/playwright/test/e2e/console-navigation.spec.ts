import type { ConsoleMessage, Page } from '@playwright/test';
import { test, expect } from './fixtures';
import { createOrganization, setSelectedOrganization } from './console-api';

const TOOLTIP_PROVIDER_ERROR = '`Tooltip` must be used within `TooltipProvider`';
const APP_READY_TIMEOUT_MS = 15000;
const ROUTE_SETTLE_TIMEOUT_MS = 500;

type NavigationTarget = {
  navTestId: string;
  title: string;
  pathPattern: RegExp;
};

const platformTargets: NavigationTarget[] = [
  { navTestId: 'nav-dashboard', title: 'Overview', pathPattern: /\/$/ },
  { navTestId: 'nav-users', title: 'Users', pathPattern: /\/users$/ },
  { navTestId: 'nav-organizations', title: 'Organizations', pathPattern: /\/organizations$/ },
  { navTestId: 'nav-cluster-runners', title: 'Runners', pathPattern: /\/runners$/ },
  { navTestId: 'nav-apps', title: 'App Catalog', pathPattern: /\/apps$/ },
];

// One entry per organization sidebar section, in sidebar order. The `LLM`
// group header carries the qualifier the `Providers` page title still spells out.
const organizationTargets: NavigationTarget[] = [
  { navTestId: 'nav-organization-overview', title: 'Overview', pathPattern: /\/organizations\/[^/]+$/ },
  { navTestId: 'nav-organization-members', title: 'Members', pathPattern: /\/organizations\/[^/]+\/members$/ },
  { navTestId: 'nav-organization-groups', title: 'Groups', pathPattern: /\/organizations\/[^/]+\/groups$/ },
  { navTestId: 'nav-organization-agents', title: 'Agents', pathPattern: /\/organizations\/[^/]+\/agents$/ },
  { navTestId: 'nav-organization-apps', title: 'Apps', pathPattern: /\/organizations\/[^/]+\/apps$/ },
  { navTestId: 'nav-organization-environments', title: 'Environments', pathPattern: /\/organizations\/[^/]+\/environments$/ },
  { navTestId: 'nav-organization-volumes', title: 'Volumes', pathPattern: /\/organizations\/[^/]+\/volumes$/ },
  { navTestId: 'nav-organization-runners', title: 'Runners', pathPattern: /\/organizations\/[^/]+\/runners$/ },
  { navTestId: 'nav-organization-private-networks', title: 'Private Networks', pathPattern: /\/organizations\/[^/]+\/private-networks$/ },
  { navTestId: 'nav-organization-private-resources', title: 'Private Resources', pathPattern: /\/organizations\/[^/]+\/private-resources$/ },
  { navTestId: 'nav-organization-egress-rules', title: 'Egress Rules', pathPattern: /\/organizations\/[^/]+\/egress-rules$/ },
  { navTestId: 'nav-organization-llm-providers', title: 'LLM Providers', pathPattern: /\/organizations\/[^/]+\/llm-providers$/ },
  { navTestId: 'nav-organization-models', title: 'Models', pathPattern: /\/organizations\/[^/]+\/models$/ },
  { navTestId: 'nav-organization-secrets', title: 'Secrets', pathPattern: /\/organizations\/[^/]+\/secrets$/ },
  { navTestId: 'nav-organization-secret-providers', title: 'Secret Providers', pathPattern: /\/organizations\/[^/]+\/secret-providers$/ },
  { navTestId: 'nav-organization-image-pull-secrets', title: 'Image Pull Secrets', pathPattern: /\/organizations\/[^/]+\/image-pull-secrets$/ },
  { navTestId: 'nav-organization-threads', title: 'Threads', pathPattern: /\/organizations\/[^/]+\/threads$/ },
  { navTestId: 'nav-organization-instances', title: 'Instances', pathPattern: /\/organizations\/[^/]+\/instances$/ },
  { navTestId: 'nav-organization-workloads', title: 'Workloads', pathPattern: /\/organizations\/[^/]+\/workloads$/ },
  { navTestId: 'nav-organization-sandboxes', title: 'Sandboxes', pathPattern: /\/organizations\/[^/]+\/sandboxes$/ },
  { navTestId: 'nav-organization-storage', title: 'Provisioned Storage', pathPattern: /\/organizations\/[^/]+\/storage$/ },
  { navTestId: 'nav-organization-usage', title: 'Usage', pathPattern: /\/organizations\/[^/]+\/usage$/ },
];

const ORGANIZATION_GROUP_TEST_IDS = [
  'nav-group-organization',
  'nav-group-agents-and-apps',
  'nav-group-runtime',
  'nav-group-networking',
  'nav-group-llm',
  'nav-group-credentials',
  'nav-group-operations',
];

// Superseded paths that must land on the canonical one.
const redirects: Array<{ from: string; to: RegExp }> = [
  { from: 'activity', to: /\/workloads$/ },
  { from: 'activity/workloads', to: /\/workloads$/ },
  { from: 'activity/storage', to: /\/storage$/ },
  { from: 'activity/threads', to: /\/threads$/ },
  { from: 'activity/usage', to: /\/usage$/ },
  { from: 'monitoring', to: /\/workloads$/ },
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
  await test.step(`assert no crash signals after ${target.title}`, async () => {
    await new Promise((resolve) => setTimeout(resolve, ROUTE_SETTLE_TIMEOUT_MS));
    expect(crashSignals, `unexpected browser crash signals after opening ${target.title}`).toEqual([]);
  });
}

async function expectShellReady(page: Page, target: NavigationTarget): Promise<void> {
  await expect(page).toHaveURL(target.pathPattern, { timeout: APP_READY_TIMEOUT_MS });
  await expect(page.getByTestId('console-sidebar')).toBeVisible({ timeout: APP_READY_TIMEOUT_MS });
  await expect(page.getByTestId('page-title')).toHaveText(target.title, { timeout: APP_READY_TIMEOUT_MS });
}

async function setClusterContext(page: Page): Promise<void> {
  await page.evaluate(() => {
    window.localStorage.setItem('console.contextMode', JSON.stringify({ mode: 'cluster' }));
    window.localStorage.removeItem('console.selectedOrganization');
  });
}

async function openNavigationTarget(page: Page, target: NavigationTarget, crashSignals: string[]): Promise<void> {
  await page.getByTestId(target.navTestId).click();
  await expectShellReady(page, target);
  await expectNoCrashSignals(crashSignals, target);
}

test.describe('console navigation', { tag: ['@svc_console', '@svc_gateway', '@smoke'] }, () => {
  test('opens every platform sidebar section without browser crashes', async ({ page }) => {
    const crashSignals = collectCrashSignals(page);

    await setClusterContext(page);
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

  test('collapses a sidebar group and keeps it collapsed across a reload', async ({ page }) => {
    const organizationId = await createOrganization(page, `e2e-org-nav-collapse-${Date.now()}`);
    await setSelectedOrganization(page, organizationId);

    await page.goto(`/organizations/${organizationId}`);
    await expectShellReady(page, organizationTargets[0]);

    for (const groupTestId of ORGANIZATION_GROUP_TEST_IDS) {
      await expect(page.getByTestId(groupTestId)).toBeVisible({ timeout: APP_READY_TIMEOUT_MS });
    }

    const operationsGroup = page.getByTestId('nav-group-operations');
    const usageLink = page.getByTestId('nav-organization-usage');
    await expect(usageLink).toBeVisible();

    await operationsGroup.click();
    await expect(usageLink).toBeHidden();
    await expect(operationsGroup).toHaveAttribute('aria-expanded', 'false');

    await page.reload();
    await expect(page.getByTestId('console-sidebar')).toBeVisible({ timeout: APP_READY_TIMEOUT_MS });
    await expect(page.getByTestId('nav-group-operations')).toHaveAttribute('aria-expanded', 'false');
    await expect(page.getByTestId('nav-organization-usage')).toBeHidden();

    await page.getByTestId('nav-group-operations').click();
    await expect(page.getByTestId('nav-organization-usage')).toBeVisible();
  });

  test('redirects superseded operations paths to their canonical path', async ({ page }) => {
    const organizationId = await createOrganization(page, `e2e-org-nav-redirect-${Date.now()}`);
    await setSelectedOrganization(page, organizationId);

    for (const redirect of redirects) {
      await page.goto(`/organizations/${organizationId}/${redirect.from}`);
      await expect(page).toHaveURL(redirect.to, { timeout: APP_READY_TIMEOUT_MS });
    }
  });
});
