import { create, fromBinary, toBinary, toJson } from '@bufbuild/protobuf';
import type { DescMessage, MessageShape } from '@bufbuild/protobuf';
import type { Page } from '@playwright/test';
import { readOidcSession } from './oidc-helpers';
import {
  AgentAvailability,
  CreateAgentRequestSchema,
  CreateAgentResponseSchema,
} from '../../src/gen/agynio/api/agents/v1/agents_pb';
import { ListSpansOrderBy } from '../../src/gen/agynio/api/tracing/v1/tracing_pb';

const USERS_GATEWAY_PATH = '/api/agynio.api.gateway.v1.UsersGateway';
const ORGS_GATEWAY_PATH = '/api/agynio.api.gateway.v1.OrganizationsGateway';
const LLM_GATEWAY_PATH = '/api/agynio.api.gateway.v1.LLMGateway';
const AGENTS_GATEWAY_PATH = '/api/agynio.api.gateway.v1.AgentsGateway';
const THREADS_GATEWAY_PATH = '/api/agynio.api.gateway.v1.ThreadsGateway';
const TRACING_GATEWAY_PATH = '/api/agynio.api.gateway.v1.TracingGateway';
const RUNNERS_GATEWAY_PATH = '/api/agynio.api.gateway.v1.RunnersGateway';

type IdentityWire = {
  meta?: { id?: string };
};

type GetMeResponseWire = {
  user?: IdentityWire;
};

type CreateOrganizationResponseWire = {
  organization?: { id?: string };
};

type CreateAPITokenResponseWire = {
  plaintextToken?: string;
  token?: { id?: string };
};

type CreateLLMProviderResponseWire = {
  provider?: { meta?: { id?: string } };
};

type CreateModelResponseWire = {
  model?: { meta?: { id?: string } };
};

type CreateAgentResponseWire = {
  agent?: { meta?: { id?: string } };
};

type CreateAgentOptions = {
  organizationId: string;
  name: string;
  model: string;
  image: string;
  initImage: string;
  description?: string;
  role?: string;
  configuration?: string;
};

type CreateAgentPayload = Omit<CreateAgentOptions, 'description' | 'role' | 'configuration'> & {
  availability: AgentAvailability.INTERNAL;
  description: string;
  role: string;
  configuration: string;
};

type CreateEnvResponseWire = {
  env?: { meta?: { id?: string } };
};

type CreateMcpResponseWire = {
  mcp?: { meta?: { id?: string } };
};

type CreateThreadResponseWire = {
  thread?: { id?: string };
};

type SendMessageResponseWire = {
  message?: { id?: string; createdAt?: string };
};

type SpanWire = {
  traceId?: string;
  name?: string;
};

type AttributeValueWire = {
  stringValue?: string;
};

type ResourceAttributeWire = {
  key?: string;
  value?: AttributeValueWire;
};

type ResourceWire = {
  attributes?: ResourceAttributeWire[];
};

type ContainerWire = {
  name?: string;
  status?: string | number;
};

type WorkloadWire = {
  meta?: { id?: string };
  containers?: ContainerWire[];
};

type ScopeSpansWire = {
  spans?: SpanWire[];
};

type ResourceSpansWire = {
  resource?: ResourceWire;
  scopeSpans?: ScopeSpansWire[];
};

export type ListSpansResponseWire = {
  resourceSpans?: ResourceSpansWire[];
  nextPageToken?: string;
};

export type TraceSummaryResponseWire = {
  status?: string | number;
  countsByName?: Record<string, number | string>;
  countsByStatus?: Record<string, number | string>;
  totalSpans?: number | string;
};

export type ListWorkloadsByThreadResponseWire = {
  workloads?: WorkloadWire[];
  nextPageToken?: string;
};

function resolveBaseUrl(): string {
  const baseUrl = process.env.E2E_BASE_URL;
  if (!baseUrl) {
    throw new Error('E2E_BASE_URL is required to run e2e tests.');
  }
  return baseUrl;
}

function buildRpcUrl(servicePath: string, method: string): string {
  return new URL(`${servicePath}/${method}`, resolveBaseUrl()).toString();
}

const CONNECT_JSON_HEADERS = {
  'Connect-Protocol-Version': '1',
  'Content-Type': 'application/json',
};

const CONNECT_PROTO_HEADERS = {
  'Connect-Protocol-Version': '1',
  'Content-Type': 'application/proto',
};

type PostConnectProtoOptions<InputSchema extends DescMessage, OutputSchema extends DescMessage> = {
  encoding: 'proto';
  inputSchema: InputSchema;
  outputSchema: OutputSchema;
};

function formatCreateAgentDebugPayload(payload: unknown): string {
  return JSON.stringify(payload);
}

async function postConnect<T>(
  page: Page,
  servicePath: string,
  method: string,
  payload: Record<string, unknown> | MessageShape<DescMessage>,
  options?: PostConnectProtoOptions<DescMessage, DescMessage>,
): Promise<T> {
  const session = await readOidcSession(page);
  const accessToken = session?.accessToken;
  if (!accessToken) {
    throw new Error('OIDC session missing access token.');
  }

  if (options?.encoding === 'proto') {
    const message = payload as MessageShape<typeof options.inputSchema>;
    const requestBody = Buffer.from(toBinary(options.inputSchema, message));
    const response = await page.context().request.post(buildRpcUrl(servicePath, method), {
      data: requestBody,
      headers: { ...CONNECT_PROTO_HEADERS, Authorization: `Bearer ${accessToken}` },
    });
    if (!response.ok()) {
      const body = await response.text();
      if (method === 'CreateAgent') {
        console.error(`Connect RPC ${method} request JSON: ${formatCreateAgentDebugPayload(toJson(options.inputSchema, message))}`);
        console.error(`Connect RPC ${method} request proto hex: ${requestBody.toString('hex')}`);
      }
      throw new Error(`Connect RPC ${servicePath}/${method} failed (${response.status()}): ${body}`);
    }
    const responseBody = Buffer.from(await response.body());
    return fromBinary(options.outputSchema, responseBody) as T;
  }

  const response = await page.context().request.post(buildRpcUrl(servicePath, method), {
    data: payload,
    headers: { ...CONNECT_JSON_HEADERS, Authorization: `Bearer ${accessToken}` },
  });
  if (!response.ok()) {
    const body = await response.text();
    if (method === 'CreateAgent') {
      console.error(`Connect RPC ${method} request JSON: ${formatCreateAgentDebugPayload(payload)}`);
    }
    throw new Error(`Connect RPC ${servicePath}/${method} failed (${response.status()}): ${body}`);
  }
  return (await response.json()) as T;
}

async function postConnectWithToken<T>(
  page: Page,
  servicePath: string,
  method: string,
  payload: Record<string, unknown>,
  token: string,
): Promise<T> {
  const response = await page.context().request.post(buildRpcUrl(servicePath, method), {
    data: payload,
    headers: {
      Authorization: `Bearer ${token}`,
      'Connect-Protocol-Version': '1',
      'Content-Type': 'application/json',
    },
  });
  if (!response.ok()) {
    const body = await response.text();
    throw new Error(`Connect RPC ${servicePath}/${method} failed (${response.status()}): ${body}`);
  }
  return (await response.json()) as T;
}

export async function getIdentityId(page: Page): Promise<string> {
  const response = await postConnect<GetMeResponseWire>(page, USERS_GATEWAY_PATH, 'GetMe', {});
  const identityId = response.user?.meta?.id;
  if (!identityId) {
    throw new Error('GetMe response missing identity id.');
  }
  return identityId;
}

export async function createOrganization(page: Page, name: string): Promise<string> {
  const response = await postConnect<CreateOrganizationResponseWire>(page, ORGS_GATEWAY_PATH, 'CreateOrganization', {
    name,
  });
  const organizationId = response.organization?.id;
  if (!organizationId) {
    throw new Error('CreateOrganization response missing organization id.');
  }
  return organizationId;
}

export async function createApiToken(page: Page, name: string): Promise<string> {
  const response = await postConnect<CreateAPITokenResponseWire>(page, USERS_GATEWAY_PATH, 'CreateAPIToken', {
    name,
  });
  const token = response.plaintextToken;
  if (!token) {
    throw new Error('CreateAPIToken response missing plaintext token.');
  }
  return token;
}

export async function createLlmProvider(page: Page, params: {
  organizationId: string;
  name: string;
  endpoint: string;
  protocol: string;
  authMethod: string;
  token: string;
}): Promise<string> {
  const response = await postConnect<CreateLLMProviderResponseWire>(page, LLM_GATEWAY_PATH, 'CreateLLMProvider', {
    organizationId: params.organizationId,
    name: params.name,
    endpoint: params.endpoint,
    protocol: params.protocol,
    authMethod: params.authMethod,
    token: params.token,
  });
  const providerId = response.provider?.meta?.id;
  if (!providerId) {
    throw new Error('CreateLLMProvider response missing provider id.');
  }
  return providerId;
}

export async function createModel(page: Page, params: {
  organizationId: string;
  llmProviderId: string;
  name: string;
  remoteName: string;
}): Promise<string> {
  const response = await postConnect<CreateModelResponseWire>(page, LLM_GATEWAY_PATH, 'CreateModel', {
    organizationId: params.organizationId,
    llmProviderId: params.llmProviderId,
    name: params.name,
    remoteName: params.remoteName,
  });
  const modelId = response.model?.meta?.id;
  if (!modelId) {
    throw new Error('CreateModel response missing model id.');
  }
  return modelId;
}

export function buildCreateAgentPayload(params: CreateAgentOptions): CreateAgentPayload {
  const initImage = params.initImage.trim();
  if (!initImage) {
    throw new Error('initImage is required to create agents.');
  }
  return {
    organizationId: params.organizationId,
    name: params.name,
    model: params.model,
    image: params.image,
    initImage,
    availability: AgentAvailability.INTERNAL,
    role: params.role ?? 'assistant',
    description: params.description ?? '',
    configuration: params.configuration ?? '',
  };
}

export function buildCreateAgentRequestJson(params: CreateAgentOptions): unknown {
  return toJson(CreateAgentRequestSchema, create(CreateAgentRequestSchema, buildCreateAgentPayload(params)));
}

export function buildCreateAgentRequestBytes(params: CreateAgentOptions): Uint8Array {
  return toBinary(CreateAgentRequestSchema, create(CreateAgentRequestSchema, buildCreateAgentPayload(params)));
}

export async function createAgent(page: Page, params: CreateAgentOptions): Promise<string> {
  const request = create(CreateAgentRequestSchema, buildCreateAgentPayload(params));
  const response = await postConnect<CreateAgentResponseWire>(
    page,
    AGENTS_GATEWAY_PATH,
    'CreateAgent',
    request,
    {
      encoding: 'proto',
      inputSchema: CreateAgentRequestSchema,
      outputSchema: CreateAgentResponseSchema,
    },
  );
  const agentId = response.agent?.meta?.id;
  if (!agentId) {
    throw new Error('CreateAgent response missing agent id.');
  }
  return agentId;
}

export async function createAgentEnv(page: Page, params: {
  agentId: string;
  name: string;
  value: string;
}): Promise<string> {
  const response = await postConnect<CreateEnvResponseWire>(page, AGENTS_GATEWAY_PATH, 'CreateEnv', {
    agentId: params.agentId,
    name: params.name,
    value: params.value,
  });
  const envId = response.env?.meta?.id;
  if (!envId) {
    throw new Error('CreateEnv response missing env id.');
  }
  return envId;
}

export async function createMcp(page: Page, params: {
  agentId: string;
  name: string;
  image: string;
  command: string;
  description?: string;
}): Promise<string> {
  const response = await postConnect<CreateMcpResponseWire>(page, AGENTS_GATEWAY_PATH, 'CreateMcp', {
    agentId: params.agentId,
    name: params.name,
    image: params.image,
    command: params.command,
    description: params.description ?? '',
  });
  const mcpId = response.mcp?.meta?.id;
  if (!mcpId) {
    throw new Error('CreateMcp response missing MCP id.');
  }
  return mcpId;
}

export async function createMcpEnv(page: Page, params: {
  mcpId: string;
  name: string;
  value: string;
}): Promise<string> {
  const response = await postConnect<CreateEnvResponseWire>(page, AGENTS_GATEWAY_PATH, 'CreateEnv', {
    mcpId: params.mcpId,
    name: params.name,
    value: params.value,
  });
  const envId = response.env?.meta?.id;
  if (!envId) {
    throw new Error('CreateEnv response missing env id.');
  }
  return envId;
}

export async function createThread(page: Page, params: {
  organizationId: string;
  participantIds: string[];
}): Promise<string> {
  const identityId = await getIdentityId(page);
  const participants = params.participantIds.filter((id) => id !== identityId);
  const response = await postConnect<CreateThreadResponseWire>(page, THREADS_GATEWAY_PATH, 'CreateThread', {
    organizationId: params.organizationId,
    participantIds: participants,
  });
  const threadId = response.thread?.id;
  if (!threadId) {
    throw new Error('CreateThread response missing thread id.');
  }
  return threadId;
}

export async function sendThreadMessage(page: Page, params: {
  threadId: string;
  senderId: string;
  body: string;
  fileIds?: string[];
}): Promise<string> {
  const response = await postConnect<SendMessageResponseWire>(page, THREADS_GATEWAY_PATH, 'SendMessage', {
    threadId: params.threadId,
    senderId: params.senderId,
    body: params.body,
    fileIds: params.fileIds ?? [],
  });
  const messageId = response.message?.id;
  if (!messageId) {
    throw new Error('SendMessage response missing message id.');
  }
  return messageId;
}

export async function listSpans(page: Page, params: {
  organizationId: string;
  filter: Record<string, unknown>;
  pageSize?: number;
  pageToken?: string;
  orderBy?: ListSpansOrderBy | number;
}): Promise<ListSpansResponseWire> {
  return postConnect<ListSpansResponseWire>(page, TRACING_GATEWAY_PATH, 'ListSpans', {
    organizationId: params.organizationId,
    filter: params.filter,
    pageSize: params.pageSize ?? 10,
    pageToken: params.pageToken ?? '',
    orderBy: params.orderBy ?? ListSpansOrderBy.START_TIME_DESC,
  });
}

export async function getTraceSummary(page: Page, traceId: string): Promise<TraceSummaryResponseWire> {
  return postConnect<TraceSummaryResponseWire>(page, TRACING_GATEWAY_PATH, 'GetTraceSummary', {
    traceId,
  });
}

export async function listWorkloadsByThread(page: Page, params: {
  threadId: string;
  agentId?: string;
  token?: string;
}): Promise<ListWorkloadsByThreadResponseWire> {
  const payload: Record<string, unknown> = {
    threadId: params.threadId,
    pageSize: 25,
  };
  if (params.agentId) {
    payload.agentId = params.agentId;
  }
  if (params.token) {
    return postConnectWithToken<ListWorkloadsByThreadResponseWire>(
      page,
      RUNNERS_GATEWAY_PATH,
      'ListWorkloadsByThread',
      payload,
      params.token,
    );
  }
  return postConnect<ListWorkloadsByThreadResponseWire>(
    page,
    RUNNERS_GATEWAY_PATH,
    'ListWorkloadsByThread',
    payload,
  );
}
