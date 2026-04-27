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
  listWorkloadsByThread,
  listSpans,
  sendThreadMessage,
} from './gateway-api';
import type { ListSpansResponseWire } from './gateway-api';

const TEST_LLM_ENDPOINTS = {
  agn: 'https://testllm.dev/v1/org/agynio/suite/agn/responses',
  codex: 'https://testllm.dev/v1/org/agynio/suite/codex/responses',
  claude: 'https://testllm.dev/v1/org/agynio/suite/claude/messages',
} as const;
type TraceSdk = keyof typeof TEST_LLM_ENDPOINTS;

const TEST_LLM_PROTOCOLS: Record<TraceSdk, string> = {
  agn: 'PROTOCOL_RESPONSES',
  codex: 'PROTOCOL_RESPONSES',
  claude: 'PROTOCOL_ANTHROPIC_MESSAGES',
};
const TEST_LLM_TOKEN = 'test-token';
const TEST_LLM_MODEL = 'mcp-tools-test';
const AGENT_IMAGE = 'alpine:3.21';
const MCP_IMAGE = 'node:22-slim';
const INIT_IMAGE_ENV_VARS: Record<TraceSdk, string> = {
  agn: 'AGN_INIT_IMAGE',
  codex: 'CODEX_INIT_IMAGE',
  claude: 'CLAUDE_INIT_IMAGE',
};
const TRACE_DISCOVER_TIMEOUT_MS = 2 * 60_000;
const TRACE_SUMMARY_TIMEOUT_MS = 2 * 60_000;
const TRACE_POLL_INTERVAL_MS = 1_000;
const MCP_READY_TIMEOUT_MS = 90_000;
const MCP_READY_POLL_INTERVAL_MS = 1_000;

const MEMORY_MCP_COMMAND =
  `npx -y supergateway@3.4.3 --stdio "npx -y @modelcontextprotocol/server-memory@2026.1.26" --outputTransport streamableHttp --port $MCP_PORT --streamableHttpPath /mcp`;
const FILESYSTEM_MCP_COMMAND =
  `mkdir -p /test-data && printf 'hello' > /test-data/hello.txt && npx -y supergateway@3.4.3 --stdio "npx -y @modelcontextprotocol/server-filesystem@2026.1.14 /test-data" --outputTransport streamableHttp --port $MCP_PORT --streamableHttpPath /mcp`;

export const MCP_TOOLS_PROMPT =
  "Create an entity called test_project of type project with observation 'A test project', then list files in /test-data";
export const MCP_TOOLS_EXPECTED_RESPONSE =
  "I've created the entity 'test_project' (type: project) with the observation 'A test project'. The /test-data directory contains one file: hello.txt.";

type TraceCounts = {
  messageCount: number;
  llmCount: number;
  toolCount: number;
};

type ResourceSpansWire = NonNullable<ListSpansResponseWire['resourceSpans']>[number];

export type FullChainRun = {
  organizationId: string;
  messageId: string;
  runId: string;
  prompt: string;
  expectedResponse: string;
};

type FullChainRunOptions = {
  sdk?: TraceSdk;
};

function resolveInitImage(sdk: TraceSdk): string {
  const envVar = INIT_IMAGE_ENV_VARS[sdk];
  const value = process.env[envVar]?.trim() ?? '';
  if (!value) {
    throw new Error(`${envVar} is required to run tracing full-chain tests.`);
  }
  return value;
}

function resolveLlmEndpoint(sdk: TraceSdk): string {
  return TEST_LLM_ENDPOINTS[sdk];
}

function resolveLlmProtocol(sdk: TraceSdk): string {
  return TEST_LLM_PROTOCOLS[sdk];
}

function resolveRunnerToken(): string | undefined {
  const token =
    process.env.E2E_CLUSTER_ADMIN_TOKEN?.trim() ||
    process.env.CLUSTER_ADMIN_TOKEN?.trim() ||
    process.env.AGYN_API_TOKEN?.trim() ||
    '';
  return token || undefined;
}

function buildMcpName(prefix: string): string {
  return prefix;
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

function extractTraceIdFromSpans(resourceSpans: ResourceSpansWire[] | undefined): string | undefined {
  for (const resourceSpan of resourceSpans ?? []) {
    for (const scopeSpan of resourceSpan.scopeSpans ?? []) {
      for (const span of scopeSpan.spans ?? []) {
        if (span.traceId) {
          return decodeTraceId(span.traceId);
        }
      }
    }
  }
  return undefined;
}

function isContainerRunning(status: string | number | undefined): boolean {
  if (typeof status === 'number') {
    return status === 1;
  }
  if (typeof status === 'string') {
    return status.toUpperCase() === 'CONTAINER_STATUS_RUNNING' || status.toUpperCase() === 'RUNNING';
  }
  return false;
}

async function waitForMcpSidecarsReady(page: Page, params: {
  threadId: string;
  agentId: string;
}): Promise<void> {
  const start = Date.now();
  const token = resolveRunnerToken();
  while (Date.now() - start < MCP_READY_TIMEOUT_MS) {
    const response = await listWorkloadsByThread(page, {
      threadId: params.threadId,
      agentId: params.agentId,
      token,
    });
    const workloads = response.workloads ?? [];
    for (const workload of workloads) {
      const containers = workload.containers ?? [];
      const mcpContainers = containers.filter((container) => container.name?.startsWith('mcp-'));
      if (mcpContainers.length === 0) {
        continue;
      }
      const allReady = mcpContainers.every((container) => isContainerRunning(container.status));
      if (allReady) {
        return;
      }
    }
    await page.waitForTimeout(MCP_READY_POLL_INTERVAL_MS);
  }
  throw new Error(`Timed out waiting for MCP sidecars on thread ${params.threadId}.`);
}

async function waitForTraceIdByMessageId(page: Page, params: {
  organizationId: string;
  messageId: string;
  sdk: TraceSdk;
}): Promise<string> {
  const start = Date.now();
  while (Date.now() - start < TRACE_DISCOVER_TIMEOUT_MS) {
    const response = await listSpans(page, {
      organizationId: params.organizationId,
      filter: { messageId: params.messageId },
      pageSize: 1,
      orderBy: ListSpansOrderBy.START_TIME_DESC,
    });
    const messageTraceId = extractTraceIdFromSpans(response.resourceSpans);
    if (messageTraceId) {
      return messageTraceId;
    }
    await page.waitForTimeout(TRACE_POLL_INTERVAL_MS);
  }
  const envVar = INIT_IMAGE_ENV_VARS[params.sdk];
  throw new Error(
    `ListSpans(filter: { messageId: ${params.messageId} }) returned no trace id after ${TRACE_DISCOVER_TIMEOUT_MS / 1000}s. ` +
      `Check message-id correlation and ensure the ${params.sdk} agent init image is up to date (override via ${envVar}).`,
  );
}

async function waitForTraceSummary(page: Page, traceId: string): Promise<void> {
  const start = Date.now();
  let lastCounts: TraceCounts | null = null;
  let lastStatus: string | number | undefined;
  while (Date.now() - start < TRACE_SUMMARY_TIMEOUT_MS) {
    const summary = await getTraceSummary(page, encodeTraceId(traceId));
    lastStatus = summary.status;
    lastCounts = extractCounts(summary.countsByName);
    if (isTraceCompleted(summary.status)) {
      if (lastCounts.messageCount >= 1 && lastCounts.llmCount >= 2 && lastCounts.toolCount >= 2) {
        return;
      }
    }
    await page.waitForTimeout(TRACE_POLL_INTERVAL_MS);
  }
  const countsDescription = lastCounts
    ? `message=${lastCounts.messageCount}, llm=${lastCounts.llmCount}, tool=${lastCounts.toolCount}`
    : 'message=0, llm=0, tool=0';
  throw new Error(`Timed out waiting for trace summary. status=${String(lastStatus)} counts=${countsDescription}`);
}

export async function createFullChainRun(page: Page, options: FullChainRunOptions = {}): Promise<FullChainRun> {
  const sdk = options.sdk ?? 'agn';
  const identityId = await getIdentityId(page);
  const organizationId = await createOrganization(page, `e2e-tracing-${randomUUID()}`);
  const apiToken = await createApiToken(page, `e2e-tracing-token-${randomUUID()}`);

  const providerId = await createLlmProvider(page, {
    organizationId,
    name: `e2e-${sdk}-testllm-${randomUUID()}`,
    endpoint: resolveLlmEndpoint(sdk),
    protocol: resolveLlmProtocol(sdk),
    authMethod: 'AUTH_METHOD_BEARER',
    token: TEST_LLM_TOKEN,
  });

  const modelId = await createModel(page, {
    organizationId,
    llmProviderId: providerId,
    name: `e2e-model-${randomUUID()}`,
    remoteName: TEST_LLM_MODEL,
  });

  const agentId = await createAgent(page, {
    organizationId,
    name: `e2e-agent-${randomUUID()}`,
    model: modelId,
    image: AGENT_IMAGE,
    initImage: resolveInitImage(sdk),
  });

  await createAgentEnv(page, {
    agentId,
    name: 'LLM_API_TOKEN',
    value: apiToken,
  });

  const memoryMcpId = await createMcp(page, {
    agentId,
    name: buildMcpName('memory'),
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
    name: buildMcpName('filesystem'),
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

  await waitForMcpSidecarsReady(page, { threadId, agentId });

  const runId = await waitForTraceIdByMessageId(page, { organizationId, messageId, sdk });
  await waitForTraceSummary(page, runId);

  return {
    organizationId,
    messageId,
    runId,
    prompt: MCP_TOOLS_PROMPT,
    expectedResponse: MCP_TOOLS_EXPECTED_RESPONSE,
  };
}
