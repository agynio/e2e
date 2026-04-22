import { argosScreenshot } from '@argos-ci/playwright';
import type { Page } from '@playwright/test';
import { test, expect } from './fixtures';
import { Unit } from '../../src/gen/agynio/api/metering/v1/metering_pb';
import {
  createLLMProvider,
  createModel,
  createOrganization,
  queryUsage,
  recordUsage,
  setSelectedOrganization,
} from './console-api';

const TEST_LLM_ENDPOINT = 'https://testllm.dev/v1/org/agynio/suite/agn/responses';
const USAGE_POLL_TIMEOUT_MS = 180_000;
const USAGE_TEST_TIMEOUT_MS = 240_000;
const USAGE_POLL_INTERVALS_MS = [1000, 2000, 5000];
const USAGE_QUERY_LOOKBACK_MS = 24 * 60 * 60 * 1000;
const MICRO_UNITS = 1_000_000n;

type UsageBucketWire = {
  value?: string | number;
};

function buildUsageRange(): { start: string; end: string } {
  const end = new Date();
  const start = new Date(end.getTime() - USAGE_QUERY_LOOKBACK_MS);
  return { start: start.toISOString(), end: end.toISOString() };
}

function parseUsageValue(text: string | null): number | null {
  if (!text) return null;
  const normalized = text.replace(/,/g, '');
  const match = normalized.match(/\d+(?:\.\d+)?/);
  if (!match) return null;
  const value = Number(match[0]);
  return Number.isFinite(value) ? value : null;
}

function parseBucketValue(rawValue: UsageBucketWire['value']): number {
  if (typeof rawValue === 'number' && Number.isFinite(rawValue)) {
    return rawValue;
  }
  if (typeof rawValue === 'string') {
    const parsed = Number(rawValue);
    if (Number.isFinite(parsed)) {
      return parsed;
    }
  }
  throw new Error(`Unexpected usage bucket value: ${String(rawValue)}`);
}

function getUsageTotal(buckets: UsageBucketWire[] | undefined): number {
  if (!buckets?.length) return 0;
  return buckets.reduce((total, bucket) => total + parseBucketValue(bucket.value), 0);
}

async function waitForUsageReady(page: Page, organizationId: string): Promise<void> {
  await expect(async () => {
    const { start, end } = buildUsageRange();
    await queryUsage(page, {
      organizationId,
      start,
      end,
      unit: 'UNIT_TOKENS',
      granularity: 'GRANULARITY_TOTAL',
    });
  }).toPass({ timeout: USAGE_POLL_TIMEOUT_MS, intervals: USAGE_POLL_INTERVALS_MS });
}

async function waitForUsageData(page: Page, organizationId: string): Promise<void> {
  await expect(async () => {
    const { start, end } = buildUsageRange();
    const response = await queryUsage(page, {
      organizationId,
      start,
      end,
      unit: 'UNIT_TOKENS',
      granularity: 'GRANULARITY_TOTAL',
    });
    const total = getUsageTotal(response.buckets);
    if (total <= 0) {
      throw new Error('Usage metrics not populated yet.');
    }
  }).toPass({ timeout: USAGE_POLL_TIMEOUT_MS, intervals: USAGE_POLL_INTERVALS_MS });
}

async function seedUsageRecords(organizationId: string): Promise<void> {
  const timestamp = new Date();
  await recordUsage(organizationId, [
    { labels: { kind: 'input' }, unit: Unit.TOKENS, value: 1200n * MICRO_UNITS, timestamp },
    { labels: { kind: 'cached' }, unit: Unit.TOKENS, value: 300n * MICRO_UNITS, timestamp },
    { labels: { kind: 'output' }, unit: Unit.TOKENS, value: 800n * MICRO_UNITS, timestamp },
    { labels: { kind: 'request', status: 'success' }, unit: Unit.COUNT, value: 1n * MICRO_UNITS, timestamp },
  ]);
}

test.describe('organization-usage', { tag: ['@svc_console'] }, () => {
  test('shows populated usage dashboard after LLM call', async ({ page }) => {
    test.setTimeout(USAGE_TEST_TIMEOUT_MS);
    const organizationId = await createOrganization(page, `e2e-org-usage-${Date.now()}`);
    await setSelectedOrganization(page, organizationId);
    await page.reload();
    await expect(page.getByTestId('page-title')).toBeVisible({ timeout: 15000 });

    const providerId = await createLLMProvider(page, {
      organizationId,
      endpoint: TEST_LLM_ENDPOINT,
      authMethod: 'AUTH_METHOD_X_API_KEY',
      token: 'e2e-test-token',
      protocol: 'PROTOCOL_RESPONSES',
    });

    const modelName = `e2e-model-usage-${Date.now()}`;
    await createModel(page, {
      organizationId,
      providerId,
      name: modelName,
      remoteName: 'summarize-history',
    });

    await page.goto(`/organizations/${organizationId}/models`);
    const row = page.getByTestId('organization-model-row').filter({ hasText: modelName });
    await expect(row).toBeVisible({ timeout: 15000 });

    await row.getByTestId('organization-model-test').click();
    await expect(page.getByTestId('organization-model-test-pending')).toBeVisible();
    await expect(page.getByTestId('organization-model-test-success')).toBeVisible({ timeout: 15000 });
    await page.getByTestId('organization-model-test-close').click();
    await expect(page.getByTestId('organization-model-test-dialog')).toHaveCount(0);

    await seedUsageRecords(organizationId);
    await waitForUsageData(page, organizationId);

    const usageNav = page.getByTestId('nav-organization-usage');
    await usageNav.scrollIntoViewIfNeeded();
    await expect(usageNav).toBeVisible({ timeout: 15000 });
    await usageNav.click();
    await expect(page).toHaveURL(new RegExp(`/organizations/${organizationId}/usage$`));
    await expect(page.getByTestId('organization-usage-header')).toBeVisible({ timeout: 15000 });
    await expect(page.getByTestId('organization-usage-llm-section')).toBeVisible({ timeout: 15000 });
    await expect(page.getByTestId('organization-usage-llm-input')).toContainText(/\d/, { timeout: 15000 });
    const llmUsageText = await page.getByTestId('organization-usage-llm-input').innerText();
    const llmUsageValue = parseUsageValue(llmUsageText);
    expect(llmUsageValue).not.toBeNull();
    expect(llmUsageValue).toBeGreaterThan(0);
    await expect(page.getByTestId('organization-usage-llm-daily-chart')).toBeVisible();
    await expect(page.getByTestId('organization-usage-compute-section')).toBeVisible();
    await expect(page.getByTestId('organization-usage-storage-section')).toBeVisible();
    await expect(page.getByTestId('organization-usage-platform-section')).toBeVisible();

    await argosScreenshot(page, 'organization-usage-dashboard');
  });

  test('shows empty state for range with no data', async ({ page }) => {
    test.setTimeout(USAGE_TEST_TIMEOUT_MS);
    const organizationId = await createOrganization(page, `e2e-org-usage-empty-${Date.now()}`);
    await setSelectedOrganization(page, organizationId);
    await page.reload();
    await expect(page.getByTestId('page-title')).toBeVisible({ timeout: 15000 });

    await waitForUsageReady(page, organizationId);

    const usageNav = page.getByTestId('nav-organization-usage');
    await usageNav.scrollIntoViewIfNeeded();
    await expect(usageNav).toBeVisible({ timeout: 15000 });
    await usageNav.click();
    await expect(page).toHaveURL(new RegExp(`/organizations/${organizationId}/usage$`));
    await expect(page.getByTestId('organization-usage-header')).toBeVisible({ timeout: 15000 });
    await expect(page.getByTestId('organization-usage-empty')).toBeVisible({ timeout: 20000 });
    await expect(page.getByTestId('organization-usage-llm-section')).toHaveCount(0);

    await argosScreenshot(page, 'organization-usage-empty');
  });
});
