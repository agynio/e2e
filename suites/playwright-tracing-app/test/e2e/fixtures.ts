import { ConnectError, createClient, type Interceptor } from '@connectrpc/connect';
import { create, fromJsonString, toJsonString } from '@bufbuild/protobuf';
import { createConnectTransport } from '@connectrpc/connect-web';
import { context, SpanStatusCode, trace } from '@opentelemetry/api';
import { OTLPTraceExporter } from '@opentelemetry/exporter-trace-otlp-grpc';
import { resourceFromAttributes } from '@opentelemetry/resources';
import { BasicTracerProvider, SimpleSpanProcessor } from '@opentelemetry/sdk-trace-base';
import { test as base, type Page } from '@playwright/test';
import { randomBytes, randomUUID } from 'node:crypto';
import { bytesToHex, flattenResourceSpans, getIntAttr, getStringAttr, hexToBytes } from '../../src/api/spanToEvent';
import { TracingGateway } from '../../src/gen/agynio/api/gateway/v1/tracing_pb';
import {
  GetSpanRequestSchema,
  GetSpanResponseSchema,
  GetTraceSpanTotalsRequestSchema,
  GetTraceSpanTotalsResponseSchema,
  GetTraceSummaryRequestSchema,
  GetTraceSummaryResponseSchema,
  ListSpansOrderBy,
  ListSpansRequestSchema,
  ListSpansResponseSchema,
  SpanStatus,
  TokenUsageTotalsSchema,
  TraceStatus,
} from '../../src/gen/agynio/api/tracing/v1/tracing_pb';
import {
  AnyValueSchema,
  InstrumentationScopeSchema,
  KeyValueSchema,
  type KeyValue,
} from '../../src/gen/opentelemetry/proto/common/v1/common_pb';
import { ResourceSchema } from '../../src/gen/opentelemetry/proto/resource/v1/resource_pb';
import {
  ResourceSpansSchema,
  ScopeSpansSchema,
  Span_EventSchema,
  SpanSchema,
  Status_StatusCode,
  type Span,
} from '../../src/gen/opentelemetry/proto/trace/v1/trace_pb';
import {
  ListAccessibleOrganizationsResponseSchema,
  OrganizationSchema,
} from '../../src/gen/agynio/api/organizations/v1/organizations_pb';
import { ensureMockAuthEmailStrategy, signInViaMockAuth } from './sign-in-helper';

const DEFAULT_TESTLLM_MODEL = 'simple-hello';
const DEFAULT_GATEWAY_BASE_URL = 'http://gateway-gateway.platform.svc.cluster.local:8080';
const DEFAULT_TRACING_ADDRESS = 'tracing.platform.svc.cluster.local:50051';
const FALLBACK_ORG_ID = randomUUID();
const SEED_MESSAGE_PREFIX = 'hello';
const SEED_RUN_TIMEOUT_MS = 420000;
const SPAN_START_GRACE_MS = 300000;

const SPAN_WAIT_TIMEOUT_MS = 300000;
const SPAN_WAIT_INTERVAL_MS = 2000;
const TRACE_STATUS_WAIT_TIMEOUT_MS = 300000;
const USE_MOCK_DATA = process.env.E2E_MOCK_DATA === 'true';
const USE_MOCK_AUTH = process.env.E2E_MOCK_AUTH === 'true';
const DEFAULT_ORG_NAME = 'E2E Organization';

type E2EConfig = {
  gatewayBaseUrl: string;
  identityGrpcBaseUrl?: string;
  authToken?: string;
  testllmModel: string;
  tracingAddress: string;
  organizationId: string;
};

export type SeededRun = {
  organizationId: string;
  threadId: string;
  runId: string;
  messageId: string;
  messageEventId: string;
  llmEventId: string;
  messageText: string;
  llmResponseText: string;
  status: TraceStatus;
};

type GatewayClients = ReturnType<typeof createGatewayClients>;
type TracingClient = GatewayClients['tracingClient'];
type ListSpansRequest = Parameters<TracingClient['listSpans']>[0];
type FlattenedSpan = ReturnType<typeof flattenResourceSpans>[number];

type MockSpan = {
  span: Span;
  resourceAttrs: KeyValue[];
};

type MockState = {
  run: SeededRun;
  spans: MockSpan[];
  resourceAttrs: KeyValue[];
};

type Fixtures = {
  mockAuthReady: void;
  seededRun: SeededRun;
};

const config = resolveConfig();

const authInterceptor: Interceptor = (next) => async (req) => {
  if (config.authToken) {
    req.header.set('Authorization', `Bearer ${config.authToken}`);
  }
  return next(req);
};

let mockState: MockState | null = null;

export const test = base.extend<Fixtures>({
  mockAuthReady: [
    async ({ playwright }, use) => {
      if (USE_MOCK_AUTH) {
        await use();
        return;
      }
      const request = await playwright.request.newContext();
      try {
        await ensureMockAuthEmailStrategy(request);
        await use();
      } finally {
        await request.dispose();
      }
    },
    { scope: 'worker' },
  ],
  seededRun: [
    async ({ browserName: _browserName }, runFixture) => {
      const seededRun = await seedTracingRun();
      await runFixture(seededRun);
    },
    { scope: 'worker', timeout: SEED_RUN_TIMEOUT_MS },
  ],
  page: async ({ page, mockAuthReady: _mockAuthReady, seededRun: _seededRun }, runFixture) => {
    if (USE_MOCK_DATA) {
      await setupMockRoutes(page);
    }
    page.on('console', (msg) => {
      if (msg.type() === 'error') {
        console.log('[browser-error]', msg.text());
      }
    });
    page.on('requestfailed', (request) => {
      console.log(`[request-failed] ${request.url()} — ${request.failure()?.errorText}`);
    });
    await signInViaMockAuth(page);
    await runFixture(page);
  },
});

export { expect } from '@playwright/test';

export const timelineForEvent = (context: SeededRun, eventId: string) =>
  `/${context.organizationId}/runs/${context.runId}?eventId=${encodeURIComponent(eventId)}&follow=false`;

export function formatSnippet(value: string | null | undefined): string | null {
  if (!value) return null;
  const lines = value.split(/\r?\n/);
  for (const line of lines) {
    const trimmed = line.trim();
    if (trimmed.length > 0) return trimmed;
  }
  return null;
}

function resolveConfig(): E2EConfig {
  const identityBaseUrl = resolveOptionalEnv('E2E_IDENTITY_GRPC_BASE_URL');
  const normalizedIdentityBaseUrl = identityBaseUrl ? normalizeBaseUrl(identityBaseUrl) : undefined;
  return {
    gatewayBaseUrl: normalizeBaseUrl(resolveOptionalEnv('E2E_GATEWAY_BASE_URL') ?? DEFAULT_GATEWAY_BASE_URL),
    identityGrpcBaseUrl: normalizedIdentityBaseUrl,
    authToken: resolveOptionalEnv('E2E_AUTH_TOKEN'),
    testllmModel: process.env.E2E_TESTLLM_MODEL_REMOTE_NAME ?? DEFAULT_TESTLLM_MODEL,
    tracingAddress: resolveTracingAddress(normalizedIdentityBaseUrl),
    organizationId: resolveOrganizationId(),
  };
}

function resolveOrganizationId(): string {
  return resolveOptionalEnv('E2E_ORG_ID') ?? FALLBACK_ORG_ID;
}

function resolveOptionalEnv(name: string): string | undefined {
  const value = process.env[name];
  return value && value.length > 0 ? value : undefined;
}

function normalizeBaseUrl(value: string): string {
  const url = new URL(value);
  const normalized = url.toString();
  return normalized.endsWith('/') ? normalized.slice(0, -1) : normalized;
}

function resolveTracingAddress(identityGrpcBaseUrl?: string): string {
  const explicit = resolveOptionalEnv('E2E_TRACING_ADDRESS');
  if (explicit) return explicit;
  const derived = identityGrpcBaseUrl ? deriveTracingAddress(identityGrpcBaseUrl) : null;
  return derived ?? DEFAULT_TRACING_ADDRESS;
}

function deriveTracingAddress(identityGrpcBaseUrl: string): string | null {
  let url: URL;
  try {
    url = new URL(identityGrpcBaseUrl);
  } catch (error) {
    console.warn('Failed to parse E2E_IDENTITY_GRPC_BASE_URL for tracing address derivation.', error);
    return null;
  }
  if (!url.hostname) return null;
  const hostParts = url.hostname.split('.');
  if (hostParts.length === 0) return null;
  hostParts[0] = 'tracing';
  const port = url.port || '50051';
  return `${hostParts.join('.')}:${port}`;
}

function createGatewayClients() {
  const transport = createConnectTransport({
    baseUrl: config.gatewayBaseUrl,
    interceptors: [authInterceptor],
  });

  return {
    tracingClient: createClient(TracingGateway, transport),
  };
}

type SeedTracePayload = {
  organizationId: string;
  threadId: string;
  messageId: string;
  messageText: string;
  llmResponseText: string;
  modelName: string;
};

type SeedTraceResult = {
  traceId: string;
  messageId: string;
  messageSpanId: string;
  llmSpanId: string;
};

async function seedTracingRun(): Promise<SeededRun> {
  if (USE_MOCK_DATA) {
    return seedMockRun();
  }
  const { tracingClient } = createGatewayClients();
  const now = Date.now();
  const organizationId = config.organizationId;
  const threadId = randomUUID();
  const messageId = randomUUID();
  const seedMessageText = `${SEED_MESSAGE_PREFIX}-${now}`;
  const seedLlmResponse = `E2E response for ${seedMessageText}`;

  const seededTrace = await exportSeedTrace({
    organizationId,
    threadId,
    messageId,
    messageText: seedMessageText,
    llmResponseText: seedLlmResponse,
    modelName: config.testllmModel,
  });

  const messageStartTimeMin = resolveSpanStartTimeMin(undefined);
  const traceIdBytes = hexToBytes(seededTrace.traceId);

  const messageSpan = await waitForSpan(
    tracingClient,
    {
      organizationId,
      filter: { traceId: traceIdBytes, names: ['invocation.message'], startTimeMin: messageStartTimeMin },
      pageSize: 200,
      pageToken: '',
      orderBy: ListSpansOrderBy.START_TIME_DESC,
    },
    (span) => {
      const spanId = bytesToHex(span.span.spanId);
      return spanId === seededTrace.messageSpanId;
    },
    `invocation.message span for trace ${seededTrace.traceId} (message: ${seedMessageText})`,
  );

  const runId = bytesToHex(messageSpan.span.traceId);
  const messageEventId = bytesToHex(messageSpan.span.spanId);
  const messageText = requireString(
    getStringAttr(messageSpan.span.attributes, 'agyn.message.text'),
    'Message span missing text',
  );

  const llmSpan = await waitForSpan(
    tracingClient,
    {
      organizationId,
      filter: { traceId: traceIdBytes, names: ['llm.call'] },
      pageSize: 200,
      pageToken: '',
      orderBy: ListSpansOrderBy.START_TIME_DESC,
    },
    (span) => {
      const spanId = bytesToHex(span.span.spanId);
      return span.span.name === 'llm.call' && spanId === seededTrace.llmSpanId;
    },
    `llm.call span for run ${runId}`,
  );

  const llmEventId = bytesToHex(llmSpan.span.spanId);
  const llmResponseText = requireString(
    getStringAttr(llmSpan.span.attributes, 'agyn.llm.response_text'),
    'LLM span missing response text',
  );

  const traceStatus = await waitForTraceCompletion(tracingClient, messageSpan.span.traceId);

  return {
    organizationId,
    threadId,
    runId,
    messageId,
    messageEventId,
    llmEventId,
    messageText,
    llmResponseText,
    status: traceStatus,
  };
}

function seedMockRun(): SeededRun {
  const now = Date.now();
  const organizationId = config.organizationId;
  const threadId = randomUUID();
  const messageId = randomUUID();
  const seedMessageText = `${SEED_MESSAGE_PREFIX}-${now}`;
  const seedLlmResponse = `E2E response for ${seedMessageText}`;
  const traceId = randomBytes(16).toString('hex');
  const messageSpanId = randomBytes(8).toString('hex');
  const llmSpanId = randomBytes(8).toString('hex');
  const startTime = BigInt(now) * 1_000_000n;
  const messageEndTime = startTime + 80_000_000n;
  const llmStartTime = startTime + 40_000_000n;
  const llmEndTime = startTime + 140_000_000n;

  const resourceAttrs = [
    stringAttr('agyn.organization.id', organizationId),
    stringAttr('agyn.thread.id', threadId),
    stringAttr('agyn.thread.message.id', messageId),
  ];

  const messageSpan = createSpan({
    traceId,
    spanId: messageSpanId,
    name: 'invocation.message',
    startTimeUnixNano: startTime,
    endTimeUnixNano: messageEndTime,
    attributes: [
      stringAttr('agyn.thread.id', threadId),
      stringAttr('agyn.message.role', 'user'),
      stringAttr('agyn.message.kind', 'invocation'),
      stringAttr('agyn.message.text', seedMessageText),
    ],
  });

  const llmSpan = createSpan({
    traceId,
    spanId: llmSpanId,
    name: 'llm.call',
    startTimeUnixNano: llmStartTime,
    endTimeUnixNano: llmEndTime,
    attributes: [
      stringAttr('agyn.thread.id', threadId),
      stringAttr('agyn.llm.response_text', seedLlmResponse),
      stringAttr('gen_ai.system', 'testllm'),
      stringAttr('gen_ai.request.model', config.testllmModel),
      stringAttr('gen_ai.response.finish_reason', 'stop'),
      intAttr('gen_ai.usage.input_tokens', 5),
      intAttr('gen_ai.usage.output_tokens', 7),
    ],
    events: [
      createSpanEvent({
        name: 'agyn.llm.context_item',
        timeUnixNano: llmStartTime + 10_000_000n,
        attributes: [
          stringAttr('agyn.context.role', 'user'),
          stringAttr('agyn.context.text', seedMessageText),
          stringAttr('agyn.context.is_new', 'true'),
          intAttr('agyn.context.size_bytes', seedMessageText.length),
        ],
      }),
    ],
  });

  const seededRun: SeededRun = {
    organizationId,
    threadId,
    runId: traceId,
    messageId,
    messageEventId: messageSpanId,
    llmEventId: llmSpanId,
    messageText: seedMessageText,
    llmResponseText: seedLlmResponse,
    status: TraceStatus.COMPLETED,
  };

  mockState = {
    run: seededRun,
    spans: [
      { span: messageSpan, resourceAttrs },
      { span: llmSpan, resourceAttrs },
    ],
    resourceAttrs,
  };

  return seededRun;
}

async function exportSeedTrace(payload: SeedTracePayload): Promise<SeedTraceResult> {
  const exporter = new OTLPTraceExporter({
    url: resolveOtlpEndpoint(config.tracingAddress),
  });
  const previousProvider = trace.getTracerProvider();
  const resourceAttributes: Record<string, string> = {
    'agyn.organization.id': payload.organizationId,
    'agyn.thread.id': payload.threadId,
  };
  // Tracing ingest requires identity verification for message ids.
  if (process.env.E2E_SEED_INCLUDE_MESSAGE_ID === 'true') {
    resourceAttributes['agyn.thread.message.id'] = payload.messageId;
  }
  const resource = resourceFromAttributes(resourceAttributes);
  const provider = new BasicTracerProvider({
    spanProcessors: [new SimpleSpanProcessor(exporter)],
    resource,
  });
  trace.setGlobalTracerProvider(provider);

  const tracer = provider.getTracer('tracing-app-e2e');
  const messageSpan = tracer.startSpan('invocation.message', {
    attributes: {
      'agyn.thread.id': payload.threadId,
      'agyn.message.role': 'user',
      'agyn.message.kind': 'invocation',
      'agyn.message.text': payload.messageText,
    },
  });
  messageSpan.setStatus({ code: SpanStatusCode.OK });

  const llmSpan = tracer.startSpan(
    'llm.call',
    {
      attributes: {
        'agyn.thread.id': payload.threadId,
        'agyn.llm.response_text': payload.llmResponseText,
        'gen_ai.system': 'testllm',
        'gen_ai.request.model': payload.modelName,
        'gen_ai.response.finish_reason': 'stop',
        'gen_ai.usage.input_tokens': 5,
        'gen_ai.usage.output_tokens': 7,
      },
    },
    trace.setSpan(context.active(), messageSpan),
  );

  llmSpan.addEvent('agyn.llm.context_item', {
    'agyn.context.role': 'user',
    'agyn.context.text': payload.messageText,
    'agyn.context.is_new': 'true',
    'agyn.context.size_bytes': payload.messageText.length,
  });
  llmSpan.setStatus({ code: SpanStatusCode.OK });
  llmSpan.end();
  messageSpan.end();

  await provider.forceFlush();
  await provider.shutdown();
  trace.setGlobalTracerProvider(previousProvider);

  return {
    traceId: messageSpan.spanContext().traceId,
    messageId: payload.messageId,
    messageSpanId: messageSpan.spanContext().spanId,
    llmSpanId: llmSpan.spanContext().spanId,
  };
}

function resolveOtlpEndpoint(address: string): string {
  const trimmed = address.trim();
  if (trimmed.startsWith('http://') || trimmed.startsWith('https://')) {
    return trimmed;
  }
  return `http://${trimmed}`;
}

function resolveBaseUrl(): string {
  const baseUrl = process.env.E2E_BASE_URL;
  if (!baseUrl) {
    throw new Error('E2E_BASE_URL is required to run e2e tests.');
  }
  return baseUrl;
}

function stripTrailingSlash(value: string): string {
  return value.replace(/\/+$/, '');
}

function resolveMockOidcConfig(): { authority: string; clientId: string; scope: string } {
  const authority = stripTrailingSlash(
    process.env.E2E_OIDC_AUTHORITY ?? new URL('/mock-oidc', resolveBaseUrl()).toString(),
  );
  const clientId = process.env.E2E_OIDC_CLIENT_ID ?? 'tracing-app-e2e';
  const scope = process.env.E2E_OIDC_SCOPE ?? 'openid profile email';
  return { authority, clientId, scope };
}

function resolveMockEnvConfig(): Record<string, string> {
  const oidc = resolveMockOidcConfig();
  return {
    API_BASE_URL: process.env.E2E_API_BASE_URL ?? '/api',
    OIDC_AUTHORITY: oidc.authority,
    OIDC_CLIENT_ID: oidc.clientId,
    OIDC_SCOPE: oidc.scope,
  };
}

async function setupMockRoutes(page: Page): Promise<void> {
  const envConfig = resolveMockEnvConfig();
  const envPayload = `window.__ENV__ = ${JSON.stringify(envConfig)};`;

  await page.route('**/env.js', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/javascript',
      body: envPayload,
    });
  });

  await page.route('**/agynio.api.gateway.v1.TracingGateway/*', async (route) => {
    const request = route.request();
    if (request.method() === 'OPTIONS') {
      await route.fulfill({ status: 204, body: '' });
      return;
    }
    const url = new URL(request.url());
    const method = url.pathname.split('/').filter(Boolean).pop() ?? '';
    const body = request.postData() ?? '{}';
    const response = handleTracingGateway(method, body);
    await route.fulfill(response);
  });

  await page.route('**/agynio.api.gateway.v1.OrganizationsGateway/*', async (route) => {
    const request = route.request();
    if (request.method() === 'OPTIONS') {
      await route.fulfill({ status: 204, body: '' });
      return;
    }
    const url = new URL(request.url());
    const method = url.pathname.split('/').filter(Boolean).pop() ?? '';
    const body = request.postData() ?? '{}';
    const response = handleOrganizationsGateway(method, body);
    await route.fulfill(response);
  });
}

function handleTracingGateway(method: string, body: string): { status: number; contentType: string; body: string } {
  const state = requireMockState();
  switch (method) {
    case 'ListSpans': {
      const request = fromJsonString(ListSpansRequestSchema, body);
      const spans = filterMockSpans(request);
      const resourceSpans = buildResourceSpans(spans, state.resourceAttrs);
      const response = create(ListSpansResponseSchema, {
        resourceSpans,
        nextPageToken: '',
      });
      return jsonResponse(ListSpansResponseSchema, response);
    }
    case 'GetSpan': {
      const request = fromJsonString(GetSpanRequestSchema, body);
      const spanId = bytesToHex(request.spanId ?? new Uint8Array());
      const match = state.spans.find((item) => bytesToHex(item.span.spanId) === spanId);
      const resourceSpans = match ? buildResourceSpans([match], state.resourceAttrs) : [];
      const response = create(GetSpanResponseSchema, { resourceSpans });
      return jsonResponse(GetSpanResponseSchema, response);
    }
    case 'GetTraceSummary': {
      const request = fromJsonString(GetTraceSummaryRequestSchema, body);
      const traceId = bytesToHex(request.traceId ?? new Uint8Array());
      const spans = traceId ? state.spans.filter((item) => bytesToHex(item.span.traceId) === traceId) : state.spans;
      const response = buildTraceSummaryResponse(spans);
      return jsonResponse(GetTraceSummaryResponseSchema, response);
    }
    case 'GetTraceSpanTotals': {
      const request = fromJsonString(GetTraceSpanTotalsRequestSchema, body);
      const spans = filterTotalsSpans(state.spans, request.names, request.statuses);
      const response = buildTraceTotalsResponse(spans);
      return jsonResponse(GetTraceSpanTotalsResponseSchema, response);
    }
    default:
      return { status: 404, contentType: 'text/plain', body: 'Unknown TracingGateway method' };
  }
}

function handleOrganizationsGateway(
  method: string,
  _body: string,
): { status: number; contentType: string; body: string } {
  const state = requireMockState();
  switch (method) {
    case 'ListAccessibleOrganizations': {
      const organization = create(OrganizationSchema, {
        id: state.run.organizationId,
        name: DEFAULT_ORG_NAME,
      });
      const response = create(ListAccessibleOrganizationsResponseSchema, {
        organizations: [organization],
      });
      return jsonResponse(ListAccessibleOrganizationsResponseSchema, response);
    }
    default:
      return { status: 404, contentType: 'text/plain', body: 'Unknown OrganizationsGateway method' };
  }
}

function jsonResponse<T>(schema: Parameters<typeof toJsonString>[0], message: T) {
  return {
    status: 200,
    contentType: 'application/json',
    body: toJsonString(schema, message),
  };
}

function requireMockState(): MockState {
  if (!mockState) {
    throw new Error('Mock state is not initialized.');
  }
  return mockState;
}

function filterMockSpans(request: ListSpansRequest): MockSpan[] {
  const state = requireMockState();
  const filter = request.filter;
  let spans = [...state.spans];

  if (filter?.traceId && filter.traceId.length > 0) {
    const traceId = bytesToHex(filter.traceId);
    spans = spans.filter((item) => bytesToHex(item.span.traceId) === traceId);
  }
  if (filter?.messageId) {
    spans = filter.messageId === state.run.messageId ? spans : [];
  }
  const names = filter?.names && filter.names.length > 0 ? filter.names : filter?.name ? [filter.name] : [];
  if (names.length > 0) {
    const nameSet = new Set(names);
    spans = spans.filter((item) => nameSet.has(item.span.name));
  }
  if (filter?.startTimeMin && filter.startTimeMin > 0n) {
    spans = spans.filter((item) => item.span.startTimeUnixNano >= filter.startTimeMin);
  }
  if (filter?.startTimeMax && filter.startTimeMax > 0n) {
    spans = spans.filter((item) => item.span.startTimeUnixNano <= filter.startTimeMax);
  }
  if (typeof filter?.inProgress === 'boolean') {
    spans = spans.filter((item) => (item.span.endTimeUnixNano === 0n) === filter.inProgress);
  }
  if (filter?.statuses && filter.statuses.length > 0) {
    const statusSet = new Set(filter.statuses);
    spans = spans.filter((item) => statusSet.has(resolveSpanStatus(item.span)));
  }

  spans.sort((a, b) => {
    const diff = a.span.startTimeUnixNano > b.span.startTimeUnixNano ? 1 : a.span.startTimeUnixNano < b.span.startTimeUnixNano ? -1 : 0;
    return request.orderBy === ListSpansOrderBy.START_TIME_ASC ? diff : -diff;
  });

  return spans;
}

function filterTotalsSpans(spans: MockSpan[], names: string[], statuses: SpanStatus[]): MockSpan[] {
  let filtered = [...spans];
  if (names.length > 0) {
    const nameSet = new Set(names);
    filtered = filtered.filter((item) => nameSet.has(item.span.name));
  }
  if (statuses.length > 0) {
    const statusSet = new Set(statuses);
    filtered = filtered.filter((item) => statusSet.has(resolveSpanStatus(item.span)));
  }
  return filtered;
}

function resolveSpanStatus(span: Span): SpanStatus {
  if (span.status?.code === Status_StatusCode.ERROR) return SpanStatus.ERROR;
  if (span.endTimeUnixNano === 0n) return SpanStatus.RUNNING;
  return SpanStatus.OK;
}

function buildTraceSummaryResponse(spans: MockSpan[]) {
  if (spans.length === 0) {
    return create(GetTraceSummaryResponseSchema, {
      status: TraceStatus.UNSPECIFIED,
      firstSpanStartTime: 0n,
      lastSpanStartTime: 0n,
      lastSpanEndTime: 0n,
      countsByName: {},
      countsByStatus: {},
      totalSpans: 0n,
    });
  }

  const startTimes = spans.map((item) => item.span.startTimeUnixNano);
  const endTimes = spans.map((item) => item.span.endTimeUnixNano);
  const firstSpanStartTime = startTimes.reduce((min, value) => (value < min ? value : min), startTimes[0]);
  const lastSpanStartTime = startTimes.reduce((max, value) => (value > max ? value : max), startTimes[0]);
  const lastSpanEndTime = endTimes.reduce((max, value) => (value > max ? value : max), endTimes[0]);

  const countsByName: Record<string, bigint> = {};
  const countsByStatus: Record<string, bigint> = {};
  for (const item of spans) {
    countsByName[item.span.name] = (countsByName[item.span.name] ?? 0n) + 1n;
    const statusKey = resolveSpanStatus(item.span) === SpanStatus.ERROR
      ? 'SPAN_STATUS_ERROR'
      : resolveSpanStatus(item.span) === SpanStatus.RUNNING
        ? 'SPAN_STATUS_RUNNING'
        : 'SPAN_STATUS_OK';
    countsByStatus[statusKey] = (countsByStatus[statusKey] ?? 0n) + 1n;
  }

  return create(GetTraceSummaryResponseSchema, {
    status: TraceStatus.COMPLETED,
    firstSpanStartTime,
    lastSpanStartTime,
    lastSpanEndTime,
    countsByName,
    countsByStatus,
    totalSpans: BigInt(spans.length),
  });
}

function buildTraceTotalsResponse(spans: MockSpan[]) {
  let inputTokens = 0;
  let outputTokens = 0;
  let cachedTokens = 0;
  let reasoningTokens = 0;
  for (const item of spans) {
    inputTokens += getIntAttr(item.span.attributes, 'gen_ai.usage.input_tokens') ?? 0;
    outputTokens += getIntAttr(item.span.attributes, 'gen_ai.usage.output_tokens') ?? 0;
    cachedTokens += getIntAttr(item.span.attributes, 'gen_ai.usage.cache_read.input_tokens') ?? 0;
    reasoningTokens += getIntAttr(item.span.attributes, 'agyn.usage.reasoning_tokens') ?? 0;
  }
  const totalTokens = inputTokens + outputTokens;
  const tokenUsage = create(TokenUsageTotalsSchema, {
    inputTokens: BigInt(inputTokens),
    outputTokens: BigInt(outputTokens),
    cacheReadInputTokens: BigInt(cachedTokens),
    reasoningTokens: BigInt(reasoningTokens),
    totalTokens: BigInt(totalTokens),
  });
  return create(GetTraceSpanTotalsResponseSchema, {
    spanCount: BigInt(spans.length),
    tokenUsage,
  });
}

function buildResourceSpans(spans: MockSpan[], resourceAttrs: KeyValue[]) {
  if (spans.length === 0) return [];
  const spanList = spans.map((item) => item.span);
  const resource = create(ResourceSchema, { attributes: resourceAttrs });
  const scope = create(InstrumentationScopeSchema, { name: 'tracing-app-e2e' });
  const scopeSpans = create(ScopeSpansSchema, { scope, spans: spanList });
  return [create(ResourceSpansSchema, { resource, scopeSpans: [scopeSpans] })];
}

function createSpan(args: {
  traceId: string;
  spanId: string;
  name: string;
  startTimeUnixNano: bigint;
  endTimeUnixNano: bigint;
  attributes?: KeyValue[];
  events?: ReturnType<typeof createSpanEvent>[];
}): Span {
  return create(SpanSchema, {
    traceId: hexToBytes(args.traceId),
    spanId: hexToBytes(args.spanId),
    name: args.name,
    startTimeUnixNano: args.startTimeUnixNano,
    endTimeUnixNano: args.endTimeUnixNano,
    attributes: args.attributes ?? [],
    events: args.events ?? [],
  });
}

function createSpanEvent(args: { name: string; timeUnixNano: bigint; attributes: KeyValue[] }) {
  return create(Span_EventSchema, {
    name: args.name,
    timeUnixNano: args.timeUnixNano,
    attributes: args.attributes,
  });
}

function stringAttr(key: string, value: string): KeyValue {
  return create(KeyValueSchema, {
    key,
    value: create(AnyValueSchema, { value: { case: 'stringValue', value } }),
  });
}

function intAttr(key: string, value: number): KeyValue {
  return create(KeyValueSchema, {
    key,
    value: create(AnyValueSchema, { value: { case: 'intValue', value: BigInt(value) } }),
  });
}

async function waitForSpan(
  tracingClient: TracingClient,
  request: ListSpansRequest,
  predicate: (span: FlattenedSpan) => boolean,
  label: string,
): Promise<FlattenedSpan> {
  const deadline = Date.now() + SPAN_WAIT_TIMEOUT_MS;
  const requestBase = { ...request };
  let lastSummary: SpanQuerySummary | null = null;
  let lastError: string | null = null;
  while (Date.now() < deadline) {
    let pageToken = requestBase.pageToken;
    const collected: FlattenedSpan[] = [];
    let hadError = false;
    do {
      try {
        const response = await tracingClient.listSpans({ ...requestBase, pageToken });
        const spans = flattenResourceSpans(response.resourceSpans);
        collected.push(...spans);
        const match = spans.find(predicate);
        if (match) return match;
        pageToken = response.nextPageToken;
      } catch (error) {
        lastError = formatSpanError(error);
        hadError = true;
        break;
      }
    } while (pageToken);
    if (!hadError) {
      lastSummary = summarizeSpanResponse(collected);
      lastError = null;
    }
    await sleep(SPAN_WAIT_INTERVAL_MS);
  }
  const summary = formatSpanSummary(lastSummary);
  const error = lastError ?? 'none';
  let diagnostics: string | null = null;
  if (!lastError && (!lastSummary || lastSummary.spanCount === 0)) {
    try {
      diagnostics = await collectSpanDiagnostics(tracingClient, requestBase);
    } catch (diagnosticError) {
      diagnostics = `failed to collect diagnostics: ${formatSpanError(diagnosticError)}`;
    }
  }
  const diagnosticSuffix = diagnostics ? ` Diagnostics: ${diagnostics}` : '';
  throw new Error(
    `Timed out waiting for ${label}. Last listSpans summary: ${summary}. Last listSpans error: ${error}.${diagnosticSuffix}`,
  );
}

async function waitForTraceCompletion(tracingClient: TracingClient, traceId: Uint8Array): Promise<TraceStatus> {
  const deadline = Date.now() + TRACE_STATUS_WAIT_TIMEOUT_MS;
  let lastStatus: TraceStatus | null = null;
  while (Date.now() < deadline) {
    const summary = await tracingClient.getTraceSummary({ traceId });
    lastStatus = summary.status;
    if (summary.status === TraceStatus.COMPLETED || summary.status === TraceStatus.ERROR) {
      return summary.status;
    }
    await sleep(SPAN_WAIT_INTERVAL_MS);
  }
  const statusLabel = formatTraceStatus(lastStatus);
  throw new Error(
    `Timed out waiting for trace ${bytesToHex(traceId)} to complete (last status: ${statusLabel})`,
  );
}

function requireString(value: string | undefined | null, message: string): string {
  if (!value) {
    throw new Error(message);
  }
  return value;
}

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

function resolveSpanStartTimeMin(createdAt: { seconds: bigint; nanos: number } | undefined): bigint {
  if (!createdAt) {
    return msToNanos(Math.max(0, Date.now() - SPAN_START_GRACE_MS));
  }
  const graceNanos = msToNanos(SPAN_START_GRACE_MS);
  const createdAtNanos = timestampToNanos(createdAt);
  return createdAtNanos > graceNanos ? createdAtNanos - graceNanos : 0n;
}

type SpanQuerySummary = {
  spanCount: number;
  sampleNames: string[];
  sampleTraceIds: string[];
};

function summarizeSpanResponse(spans: FlattenedSpan[]): SpanQuerySummary {
  return {
    spanCount: spans.length,
    sampleNames: spans.slice(0, 5).map((span) => span.span.name),
    sampleTraceIds: spans.slice(0, 3).map((span) => bytesToHex(span.span.traceId)),
  };
}

function formatSpanSummary(summary: SpanQuerySummary | null): string {
  if (!summary) return 'none';
  const names = summary.sampleNames.length > 0 ? summary.sampleNames.join(', ') : 'none';
  const traceIds = summary.sampleTraceIds.length > 0 ? summary.sampleTraceIds.join(', ') : 'none';
  return `spanCount=${summary.spanCount}, sampleNames=[${names}], sampleTraceIds=[${traceIds}]`;
}

type SpanSummaryResult = {
  label: string;
  summary: SpanQuerySummary | null;
  error: string | null;
};

async function collectSpanDiagnostics(
  tracingClient: TracingClient,
  requestBase: ListSpansRequest,
): Promise<string> {
  const diagnosticRequests: SpanSummaryResult[] = [];
  const orderBy = requestBase.orderBy ?? ListSpansOrderBy.START_TIME_DESC;
  const baseRequest = {
    pageSize: 50,
    pageToken: '',
    orderBy,
    organizationId: requestBase.organizationId,
  };
  diagnosticRequests.push(
    await safeSpanSummary(tracingClient, 'timeWindowSummary', {
      ...baseRequest,
      filter: buildDiagnosticFilter(requestBase.filter),
    }),
  );
  diagnosticRequests.push(
    await safeSpanSummary(tracingClient, 'invocationSummary', {
      ...baseRequest,
      filter: { names: ['invocation.message'] },
    }),
  );
  const allSummary = await safeSpanSummary(tracingClient, 'allSummary', baseRequest);
  diagnosticRequests.push(allSummary);

  const parts = [`config=${formatSeedConfig()}`, `filter=${formatSpanFilter(requestBase.filter)}`];
  parts.push(...diagnosticRequests.map(formatSpanSummaryResult));

  if (allSummary.summary?.sampleTraceIds.length) {
    const traceStatuses = await summarizeTraceStatuses(tracingClient, allSummary.summary.sampleTraceIds);
    if (traceStatuses) {
      parts.push(`traceStatuses=${traceStatuses}`);
    }
  }

  return parts.join('. ');
}

async function safeSpanSummary(
  tracingClient: TracingClient,
  label: string,
  request: ListSpansRequest,
): Promise<SpanSummaryResult> {
  try {
    const summary = await summarizeListSpans(tracingClient, request);
    return { label, summary, error: null };
  } catch (error) {
    return { label, summary: null, error: formatSpanError(error) };
  }
}

async function summarizeListSpans(
  tracingClient: TracingClient,
  request: ListSpansRequest,
): Promise<SpanQuerySummary> {
  const response = await tracingClient.listSpans(request);
  const spans = flattenResourceSpans(response.resourceSpans);
  return summarizeSpanResponse(spans);
}

function formatSpanSummaryResult(result: SpanSummaryResult): string {
  if (result.error) return `${result.label}=error(${result.error})`;
  return `${result.label}=${formatSpanSummary(result.summary)}`;
}

function buildDiagnosticFilter(filter: ListSpansRequest['filter'] | undefined): ListSpansRequest['filter'] | undefined {
  if (!filter) return undefined;
  const diagnostic: ListSpansRequest['filter'] = {};
  if (filter.traceId && filter.traceId.length > 0) {
    diagnostic.traceId = filter.traceId;
  }
  if (filter.startTimeMin !== undefined) {
    diagnostic.startTimeMin = filter.startTimeMin;
  }
  if (filter.startTimeMax !== undefined) {
    diagnostic.startTimeMax = filter.startTimeMax;
  }
  if (filter.messageId) {
    diagnostic.messageId = filter.messageId;
  }
  return diagnostic;
}

function formatSpanFilter(filter: ListSpansRequest['filter'] | undefined): string {
  if (!filter) return 'none';
  const parts: string[] = [];
  if (filter.traceId && filter.traceId.length > 0) {
    parts.push(`traceId=${bytesToHex(filter.traceId)}`);
  }
  if (filter.parentSpanId && filter.parentSpanId.length > 0) {
    parts.push(`parentSpanId=${bytesToHex(filter.parentSpanId)}`);
  }
  if (filter.names && filter.names.length > 0) {
    parts.push(`names=[${filter.names.join(', ')}]`);
  }
  if (filter.name) {
    parts.push(`name=${filter.name}`);
  }
  if (filter.startTimeMin !== undefined) {
    parts.push(`startTimeMin=${filter.startTimeMin.toString()}`);
  }
  if (filter.startTimeMax !== undefined) {
    parts.push(`startTimeMax=${filter.startTimeMax.toString()}`);
  }
  if (filter.messageId) {
    parts.push(`messageId=${filter.messageId}`);
  }
  if (typeof filter.inProgress === 'boolean') {
    parts.push(`inProgress=${filter.inProgress}`);
  }
  if (filter.statuses && filter.statuses.length > 0) {
    parts.push(`statuses=[${filter.statuses.join(', ')}]`);
  }
  if (filter.kind && filter.kind !== 0) {
    parts.push(`kind=${filter.kind}`);
  }
  return parts.length ? parts.join(', ') : 'empty';
}

function formatSeedConfig(): string {
  const identityBase = config.identityGrpcBaseUrl ?? 'none';
  const seedMode = USE_MOCK_DATA ? 'mock' : 'otlp';
  return `gateway=${config.gatewayBaseUrl}, tracing=${config.tracingAddress}, identityGrpc=${identityBase}, org=${config.organizationId}, model=${config.testllmModel}, seedMode=${seedMode}`;
}

async function summarizeTraceStatuses(tracingClient: TracingClient, traceIds: string[]): Promise<string | null> {
  const uniqueIds = [...new Set(traceIds.filter((traceId) => traceId.length > 0))].slice(0, 3);
  if (uniqueIds.length === 0) return null;
  const statuses: string[] = [];
  for (const traceId of uniqueIds) {
    try {
      const summary = await tracingClient.getTraceSummary({ traceId: hexToBytes(traceId) });
      statuses.push(`${traceId}:${formatTraceStatus(summary.status)}`);
    } catch (error) {
      statuses.push(`${traceId}:error(${formatSpanError(error)})`);
    }
  }
  return statuses.join(', ');
}

function formatSpanError(error: unknown): string {
  if (error instanceof ConnectError) {
    return `${error.code}: ${error.message}`;
  }
  if (error instanceof Error) {
    return error.message;
  }
  return String(error);
}

function formatTraceStatus(status: TraceStatus | null): string {
  if (status === null) return 'unknown';
  return TraceStatus[status] ?? `unknown(${status})`;
}

function msToNanos(ms: number): bigint {
  return BigInt(ms) * 1_000_000n;
}

function timestampToNanos(timestamp: { seconds: bigint; nanos: number }): bigint {
  return timestamp.seconds * 1_000_000_000n + BigInt(timestamp.nanos);
}
