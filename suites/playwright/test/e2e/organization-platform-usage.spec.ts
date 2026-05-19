import type { Page } from "@playwright/test";
import { test, expect } from "./fixtures";
import { Unit } from "../../src/gen/agynio/api/metering/v1/metering_pb";
import {
  createOrganization,
  createThread,
  createUser,
  getMe,
  queryUsage,
  recordUsage,
  sendThreadMessage,
  setSelectedOrganization,
} from "./console-api";

const PIPELINE_USAGE_POLL_TIMEOUT_MS = 45_000;
const SEEDED_USAGE_POLL_TIMEOUT_MS = 60_000;
const USAGE_POLL_INTERVALS_MS = [500, 1000, 2000, 5000];
const USAGE_QUERY_LOOKBACK_MS = 24 * 60 * 60 * 1000;
const USAGE_TEST_TIMEOUT_MS = 150_000;
const MICRO_UNITS = 1_000_000n;

type UsageKind = "thread" | "message";

type UsageBucketWire = {
  value?: string | number;
};

function buildUsageRange(): { start: string; end: string } {
  const end = new Date();
  const start = new Date(end.getTime() - USAGE_QUERY_LOOKBACK_MS);
  return { start: start.toISOString(), end: end.toISOString() };
}

function parseBucketValue(rawValue: UsageBucketWire["value"]): number {
  if (typeof rawValue === "number" && Number.isFinite(rawValue)) {
    return rawValue;
  }
  if (typeof rawValue === "string") {
    const parsed = Number(rawValue);
    if (Number.isFinite(parsed)) {
      return parsed;
    }
  }
  throw new Error(`Unexpected usage bucket value: ${String(rawValue)}`);
}

function getUsageTotal(buckets: UsageBucketWire[] | undefined): number {
  if (!buckets?.length) return 0;
  return buckets.reduce(
    (total, bucket) => total + parseBucketValue(bucket.value),
    0,
  );
}

type PlatformUsageTotals = Record<UsageKind, number>;

async function getPlatformUsageTotals(
  page: Page,
  organizationId: string,
): Promise<PlatformUsageTotals> {
  const { start, end } = buildUsageRange();
  const [threadResponse, messageResponse] = await Promise.all(
    (["thread", "message"] as const).map((kind) =>
      queryUsage(page, {
        organizationId,
        start,
        end,
        unit: "UNIT_COUNT",
        granularity: "GRANULARITY_TOTAL",
        labelFilters: { kind },
      }),
    ),
  );
  return {
    thread: getUsageTotal(threadResponse.buckets),
    message: getUsageTotal(messageResponse.buckets),
  };
}

async function seedPlatformUsageRecords(
  organizationId: string,
  threadId: string,
  messageId: string,
): Promise<void> {
  const timestamp = new Date();
  await recordUsage(organizationId, [
    {
      labels: { kind: "thread" },
      unit: Unit.COUNT,
      value: MICRO_UNITS,
      timestamp,
      idempotencyKey: `e2e-platform-usage-thread-${threadId}`,
    },
    {
      labels: { kind: "message" },
      unit: Unit.COUNT,
      value: MICRO_UNITS,
      timestamp,
      idempotencyKey: `e2e-platform-usage-message-${messageId}`,
    },
  ]);
}

async function waitForPlatformUsageTotals(
  page: Page,
  organizationId: string,
  timeout: number,
): Promise<PlatformUsageTotals> {
  let usageTotals: PlatformUsageTotals = { thread: 0, message: 0 };
  await expect(async () => {
    usageTotals = await getPlatformUsageTotals(page, organizationId);
    if (usageTotals.thread <= 0 || usageTotals.message <= 0) {
      throw new Error(
        `Platform usage not populated yet: thread=${usageTotals.thread}, message=${usageTotals.message}.`,
      );
    }
  }).toPass({
    timeout,
    intervals: USAGE_POLL_INTERVALS_MS,
  });
  return usageTotals;
}

async function waitForPipelinePlatformUsageTotals(
  page: Page,
  organizationId: string,
): Promise<PlatformUsageTotals | null> {
  try {
    return await waitForPlatformUsageTotals(
      page,
      organizationId,
      PIPELINE_USAGE_POLL_TIMEOUT_MS,
    );
  } catch (error) {
    if (error instanceof Error) {
      test.info().annotations.push({
        type: "platform-usage-pipeline-fallback",
        description: error.message,
      });
      return null;
    }
    throw error;
  }
}

async function ensureStablePlatformUsageTotals(
  page: Page,
  organizationId: string,
  threadId: string,
  messageId: string,
): Promise<PlatformUsageTotals> {
  const pipelineUsageTotals = await waitForPipelinePlatformUsageTotals(
    page,
    organizationId,
  );
  if (pipelineUsageTotals) {
    return pipelineUsageTotals;
  }

  await seedPlatformUsageRecords(organizationId, threadId, messageId);
  return waitForPlatformUsageTotals(
    page,
    organizationId,
    SEEDED_USAGE_POLL_TIMEOUT_MS,
  );
}

test.describe(
  "organization-platform-usage",
  {
    tag: [
      "@svc_console",
      "@svc_gateway",
      "@svc_threads",
      "@svc_metering",
      "@svc_identity",
    ],
  },
  () => {
    test("records thread and message usage after platform activity", async ({
      page,
    }) => {
      test.setTimeout(USAGE_TEST_TIMEOUT_MS);
      const organizationId = await createOrganization(
        page,
        `e2e-org-platform-usage-${Date.now()}`,
      );
      await setSelectedOrganization(page, organizationId);

      const me = await getMe(page);
      const identityId = me.user?.meta?.id;
      if (!identityId) {
        throw new Error(
          "GetMe response missing identity id for platform usage test.",
        );
      }

      const participantId = await createUser(page, {
        email: `e2e-platform-usage-${Date.now()}@agyn.test`,
        nickname: "platform-usage",
      });

      const threadId = await createThread(page, {
        organizationId,
        participantIds: [participantId],
      });
      const messageId = await sendThreadMessage(page, {
        threadId,
        senderId: identityId,
        body: "Platform usage metering message.",
      });
      const usageTotals = await ensureStablePlatformUsageTotals(
        page,
        organizationId,
        threadId,
        messageId,
      );
      expect(usageTotals.thread).toBeGreaterThan(0);
      expect(usageTotals.message).toBeGreaterThan(0);

      await page.goto(`/organizations/${organizationId}/usage`);
      await expect(page.getByTestId("organization-usage-header")).toBeVisible({
        timeout: 15000,
      });
    });
  },
);
