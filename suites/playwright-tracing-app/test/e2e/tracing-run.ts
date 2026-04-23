import type { Page } from '@playwright/test';
import { randomUUID } from 'node:crypto';
import { ListSpansOrderBy, TraceStatus } from '../../src/gen/agynio/api/tracing/v1/tracing_pb';
import {
  createAgent,
  createAgentEnv,
  createApiToken,
  createLlmProvider,
  createMcp,
  createMcpEnv,
  createModel,
  createOrganization,
  createThread,
  getIdentityId,
  getTraceSummary,
  listSpans,
  sendThreadMessage,
} from './gateway-api';

const TEST_LLM_ENDPOINT = 'https://testllm.dev/v1/org/agynio/suite/agn/responses';
const TEST_LLM_TOKEN = 'test-token';
const TEST_LLM_MODEL = 'mcp-tools-test';
const AGENT_IMAGE = 'alpine:3.21';
const MCP_IMAGE = 'node:22-slim';
const DEFAULT_INIT_IMAGE = 'ghcr.io/agynio/agent-init-agn:0.4.1';
const TRACE_DISCOVER_TIMEOUT_MS = 2 * 60_000;
const TRACE_SUMMARY_TIMEOUT_MS = 2 * 60_000;
const TRACE_POLL_INTERVAL_MS = 1_000;

const MEMORY_MCP_COMMAND =
  `npx -y supergateway@3.4.3 --stdio "npx -y @modelcontextprotocol/server-memory@2026.1.26" --outputTransport streamableHttp --port $MCP_PORT --streamableHttpPath /mcp`;
const FILESYSTEM_MCP_COMMAND =
  `mkdir -p /test-data && printf 'hello' > /test-data/hello.txt && npx -y supergateway@3.4.3 --stdio "npx -y @modelcontextprotocol/server-filesystem@2026.1.14 /test-data" --outputTransport streamableHttp --port $MCP_PORT --streamableHttpPath /mcp`;

export const MCP_TOOLS_PROMPT =
  'Create an entity called test_project, then list the files in the project directory.';
export const MCP_TOOLS_EXPECTED_RESPONSE =
  "I've created the entity 'test_project' and confirmed the directory is empty.";

type TraceCounts = {
  messageCount: number;
  llmCount: number;
  toolCount: number;
};

export type FullChainRun = {
  organizationId: string;
  messageId: string;
  runId: string;
  prompt: string;
  expectedResponse: string;
};

function resolveInitImage(): string {
  const override = process.env.AGN_INIT_IMAGE?.trim();
  return override || DEFAULT_INIT_IMAGE;
}

function isHex(value: string): boolean {
  return /^[0-9a-fA-F]+$/.test(value) && value.length % 2 === 0;
}

function decodeTraceId(value: string): string {
  if (isHex(value)) {
    return value.toLowerCase();
  }
  return Buffer.from(value, 'base64').toString('hex');
}

function encodeTraceId(traceId: string): string {
  const normalized = traceId.replace(/^0x/, '');
  if (!isHex(normalized)) {
    throw new Error(`Trace id is not valid hex: ${traceId}`);
  }
  return Buffer.from(normalized, 'hex').toString('base64');
}

function parseCount(value: number | string | undefined): number {
  if (typeof value === 'number') return value;
  if (typeof value === 'string') {
    const parsed = Number.parseInt(value, 10);
    return Number.isNaN(parsed) ? 0 : parsed;
  }
  return 0;
}

function isTraceCompleted(status: string | number | undefined): boolean {
  if (typeof status === 'number') {
    return status === TraceStatus.COMPLETED;
  }
  if (typeof status === 'string') {
    const normalized = status.toUpperCase();
    return normalized === 'TRACE_STATUS_COMPLETED' || normalized === 'COMPLETED';
  }
  return false;
}

function extractCounts(counts: Record<string, number | string> | undefined): TraceCounts {
  const lookup = counts ?? {};
  return {
    messageCount: parseCount(lookup['invocation.message']),
    llmCount: parseCount(lookup['llm.call']),
    toolCount: parseCount(lookup['tool.execution']),
  };
}

async function waitForTraceIdByMessageId(page: Page, params: {
  organizationId: string;
  messageId: string;
}): Promise<string> {
  const start = Date.now();
  while (Date.now() - start < TRACE_DISCOVER_TIMEOUT_MS) {
    const response = await listSpans(page, {
      organizationId: params.organizationId,
      filter: { messageId: params.messageId },
      pageSize: 1,
      orderBy: ListSpansOrderBy.START_TIME_DESC,
    });
    for (const resourceSpan of response.resourceSpans ?? []) {
      for (const scopeSpan of resourceSpan.scopeSpans ?? []) {
        for (const span of scopeSpan.spans ?? []) {
          if (span.traceId) {
            return decodeTraceId(span.traceId);
          }
        }
      }
    }
    await page.waitForTimeout(TRACE_POLL_INTERVAL_MS);
  }
  throw new Error(`Timed out waiting for trace id for message ${params.messageId}.`);
}

async function waitForTraceSummary(page: Page, traceId: string): Promise<void> {
  const start = Date.now();
  let lastCounts: TraceCounts | null = null;
  let lastStatus: string | number | undefined;
  while (Date.now() - start < TRACE_SUMMARY_TIMEOUT_MS) {
    const summary = await getTraceSummary(page, encodeTraceId(traceId));
    lastStatus = summary.status;
    lastCounts = extractCounts(summary.countsByName);
    if (
      isTraceCompleted(summary.status) &&
      lastCounts.messageCount >= 1 &&
      lastCounts.llmCount >= 2 &&
      lastCounts.toolCount >= 2
    ) {
      return;
    }
    await page.waitForTimeout(TRACE_POLL_INTERVAL_MS);
  }
  const countsDescription = lastCounts
    ? `message=${lastCounts.messageCount}, llm=${lastCounts.llmCount}, tool=${lastCounts.toolCount}`
    : 'unavailable';
  throw new Error(`Timed out waiting for trace summary. status=${String(lastStatus)} counts=${countsDescription}`);
}

export async function createFullChainRun(page: Page): Promise<FullChainRun> {
  const identityId = await getIdentityId(page);
  const organizationId = await createOrganization(page, `e2e-tracing-${randomUUID()}`);
  const apiToken = await createApiToken(page, `e2e-tracing-token-${randomUUID()}`);

  const providerId = await createLlmProvider(page, {
    organizationId,
    name: `e2e-testllm-${randomUUID()}`,
    endpoint: TEST_LLM_ENDPOINT,
    protocol: 'PROTOCOL_RESPONSES',
    authMethod: 'AUTH_METHOD_BEARER',
    token: TEST_LLM_TOKEN,
  });

  const modelId = await createModel(page, {
    organizationId,
    providerId,
    name: `e2e-model-${randomUUID()}`,
    remoteName: TEST_LLM_MODEL,
  });

  const agentId = await createAgent(page, {
    organizationId,
    name: `e2e-agent-${randomUUID()}`,
    model: modelId,
    image: AGENT_IMAGE,
    initImage: resolveInitImage(),
  });

  await createAgentEnv(page, {
    agentId,
    name: 'LLM_API_TOKEN',
    value: apiToken,
  });

  const memoryMcpId = await createMcp(page, {
    agentId,
    name: `memory-${randomUUID()}`,
    image: MCP_IMAGE,
    command: MEMORY_MCP_COMMAND,
  });

  await createMcpEnv(page, {
    mcpId: memoryMcpId,
    name: 'MEMORY_FILE_PATH',
    value: '/tmp/memory.json',
  });

  await createMcp(page, {
    agentId,
    name: `filesystem-${randomUUID()}`,
    image: MCP_IMAGE,
    command: FILESYSTEM_MCP_COMMAND,
  });

  const threadId = await createThread(page, {
    organizationId,
    participantIds: [identityId, agentId],
  });

  const messageId = await sendThreadMessage(page, {
    threadId,
    senderId: identityId,
    body: MCP_TOOLS_PROMPT,
  });

  const runId = await waitForTraceIdByMessageId(page, { organizationId, messageId });
  await waitForTraceSummary(page, runId);

  return {
    organizationId,
    messageId,
    runId,
    prompt: MCP_TOOLS_PROMPT,
    expectedResponse: MCP_TOOLS_EXPECTED_RESPONSE,
  };
}
