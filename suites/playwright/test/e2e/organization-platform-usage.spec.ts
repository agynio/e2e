import type { Page } from "@playwright/test";
import { test, expect } from "./fixtures";
import {
  createOrganization,
  createThread,
  createUser,
  getMe,
  queryUsage,
  sendThreadMessage,
  setSelectedOrganization,
} from "./console-api";

const USAGE_POLL_TIMEOUT_MS = 180_000;
const USAGE_POLL_INTERVALS_MS = [1000, 2000, 5000];
const USAGE_QUERY_LOOKBACK_MS = 24 * 60 * 60 * 1000;

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

async function getPlatformUsageTotal(
  page: Page,
  organizationId: string,
  kind: UsageKind,
): Promise<number> {
  const { start, end } = buildUsageRange();
  const response = await queryUsage(page, {
    organizationId,
    start,
    end,
    unit: "UNIT_COUNT",
    granularity: "GRANULARITY_TOTAL",
    labelFilters: { kind },
  });
  return getUsageTotal(response.buckets);
}

async function waitForPlatformUsage(
  page: Page,
  organizationId: string,
  kind: UsageKind,
): Promise<number> {
  let usageTotal = 0;
  await expect(async () => {
    usageTotal = await getPlatformUsageTotal(page, organizationId, kind);
    if (usageTotal <= 0) {
      throw new Error(`Platform ${kind} usage not populated yet.`);
    }
  }).toPass({
    timeout: USAGE_POLL_TIMEOUT_MS,
    intervals: USAGE_POLL_INTERVALS_MS,
  });
  return usageTotal;
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
      test.setTimeout(240_000);
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
      await sendThreadMessage(page, {
        threadId,
        senderId: identityId,
        body: "Platform usage metering message.",
      });

      const threadUsageTotal = await waitForPlatformUsage(
        page,
        organizationId,
        "thread",
      );
      const messageUsageTotal = await waitForPlatformUsage(
        page,
        organizationId,
        "message",
      );
      expect(threadUsageTotal).toBeGreaterThan(0);
      expect(messageUsageTotal).toBeGreaterThan(0);

      await page.goto(`/organizations/${organizationId}/usage`);
      await expect(page.getByTestId("organization-usage-header")).toBeVisible({
        timeout: 15000,
      });
    });
  },
);
