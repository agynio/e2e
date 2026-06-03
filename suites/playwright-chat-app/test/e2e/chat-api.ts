import { create, fromBinary, toBinary, toJson } from '@bufbuild/protobuf';
import type { DescMessage, MessageShape } from '@bufbuild/protobuf';
import type { Page } from '@playwright/test';
import { AgentAvailability, CreateAgentRequestSchema, CreateAgentResponseSchema } from '../../src/gen/agynio/api/agents/v1/agents_pb';

const CHAT_GATEWAY_PATH = '/api/agynio.api.gateway.v1.ChatGateway';
const AGENTS_GATEWAY_PATH = '/api/agynio.api.gateway.v1.AgentsGateway';
const LLM_GATEWAY_PATH = '/api/agynio.api.gateway.v1.LLMGateway';
const ORGS_GATEWAY_PATH = '/api/agynio.api.gateway.v1.OrganizationsGateway';
const USERS_GATEWAY_PATH = '/api/agynio.api.gateway.v1.UsersGateway';

export const DEFAULT_TEST_INIT_IMAGE =
  process.env.E2E_AGENT_INIT_IMAGE?.trim() ||
  process.env.CODEX_INIT_IMAGE?.trim() ||
    'ghcr.io/agynio/agent-init-codex:0.13.29';
export const DEFAULT_TEST_AGENT_IMAGE = 'alpine:3.21';

const CONNECT_JSON_HEADERS = {
  'Content-Type': 'application/json',
  'Connect-Protocol-Version': '1',
};

const CONNECT_PROTO_HEADERS = {
  'Content-Type': 'application/proto',
  'Connect-Protocol-Version': '1',
};

const CHAT_STATUS_MAP = {
  open: 'CHAT_STATUS_OPEN',
  closed: 'CHAT_STATUS_CLOSED',
} satisfies Record<'open' | 'closed', string>;

const MEMBERSHIP_ROLE_MAP = {
  MEMBERSHIP_ROLE_OWNER: 'MEMBERSHIP_ROLE_OWNER',
  MEMBERSHIP_ROLE_MEMBER: 'MEMBERSHIP_ROLE_MEMBER',
} satisfies Record<MembershipRoleValue, string>;

const DEBUG_CREATE_AGENT_PAYLOAD = process.env.E2E_DEBUG_CREATE_AGENT_PAYLOAD === 'true';
const REDACTED_VALUE = '<redacted>';
const SENSITIVE_KEY_PATTERN = /token|secret|password|authorization|credential/i;

export function resolveCodexInitImage(override?: string): string {
  if (override !== undefined) {
    const trimmed = override.trim();
    if (!trimmed) {
      throw new Error('initImage is required to create chat agents.');
    }
    return trimmed;
  }
  const value = process.env.CODEX_INIT_IMAGE?.trim();
  if (value) {
    return value;
  }
  return DEFAULT_TEST_INIT_IMAGE;
}

type CreateChatResponseWire = {
  chat?: { id?: string };
};

type CreateOrganizationResponseWire = {
  organization?: { id?: string };
};

type CreateMembershipResponseWire = {
  membership?: { id?: string };
};

type GetMeResponseWire = {
  user?: { meta?: { id?: string } };
};

type ListAccessibleOrganizationsResponseWire = {
  organizations?: Array<{ id: string; name: string }>;
};

type CreateAgentResponseWire = {
  agent?: { meta?: { id?: string } };
};

type CreateEnvResponseWire = {
  env?: { meta?: { id?: string } };
};

type CreateLLMProviderResponseWire = {
  provider?: { meta?: { id?: string } };
};

type CreateModelResponseWire = {
  model?: { meta?: { id?: string } };
};

type Message = {
  id?: string;
  senderId?: string;
  body?: string;
};

type ChatSummary = {
  id?: string;
  status?: string;
};

type GetChatsResponseWire = {
  chats?: ChatSummary[];
  nextPageToken?: string;
};

type MembershipRoleValue = 'MEMBERSHIP_ROLE_OWNER' | 'MEMBERSHIP_ROLE_MEMBER';

type GetMessagesResponseWire = {
  messages?: Message[];
};

type ListAgentsResponseWire = {
  agents?: Array<{ meta?: { id?: string }; name?: string }>;
  nextPageToken?: string;
};

type BatchGetUsersResponseWire = {
  users?: Array<{ meta?: { id?: string }; name?: string; email?: string }>;
};

const CHAT_ORG_STORAGE_KEY = 'ui.organization.chat-map';
const CHAT_ORG_STORAGE_VERSION = 1;

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

type OidcStorageSnapshot = {
  accessToken: string | null;
};

type PostConnectProtoOptions<InputSchema extends DescMessage, OutputSchema extends DescMessage> = {
  encoding: 'proto';
  inputSchema: InputSchema;
  outputSchema: OutputSchema;
};

function redactPayload(value: unknown): unknown {
  if (Array.isArray(value)) {
    return value.map((entry) => redactPayload(entry));
  }
  if (!value || typeof value !== 'object') {
    return value;
  }
  const redacted: Record<string, unknown> = {};
  for (const [key, entry] of Object.entries(value)) {
    redacted[key] = SENSITIVE_KEY_PATTERN.test(key) ? REDACTED_VALUE : redactPayload(entry);
  }
  return redacted;
}

function formatDebugPayload(payload: unknown): string {
  return JSON.stringify(redactPayload(payload));
}

async function readOidcSession(page: Page): Promise<OidcStorageSnapshot | null> {
  return page.evaluate(() => {
    let storageKey: string | null = null;
    for (let i = 0; i < window.sessionStorage.length; i += 1) {
      const key = window.sessionStorage.key(i);
      if (key && key.startsWith('oidc.user:')) {
        storageKey = key;
        break;
      }
    }

    if (!storageKey) return null;
    const raw = window.sessionStorage.getItem(storageKey);
    if (!raw) return null;

    try {
      const parsed = JSON.parse(raw) as { access_token?: unknown };
      return {
        accessToken: typeof parsed.access_token === 'string' ? parsed.access_token : null,
      };
    } catch (_error) {
      return null;
    }
  });
}

async function postConnect<T>(
  page: Page,
  servicePath: string,
  method: string,
  payload: unknown,
  options?: PostConnectProtoOptions<DescMessage, DescMessage>,
): Promise<T> {
  const session = await readOidcSession(page);
  const token = session?.accessToken ?? null;
  if (options?.encoding === 'proto') {
    const headers = token ? { ...CONNECT_PROTO_HEADERS, Authorization: `Bearer ${token}` } : CONNECT_PROTO_HEADERS;
    const message = payload as MessageShape<typeof options.inputSchema>;
    const requestBody = Buffer.from(toBinary(options.inputSchema, message));
    const response = await page.context().request.post(buildRpcUrl(servicePath, method), {
      data: requestBody,
      headers,
    });
    if (!response.ok()) {
      const body = await response.text();
      if (method === 'CreateAgent') {
        console.error(`ConnectRPC ${method} request JSON: ${formatDebugPayload(toJson(options.inputSchema, message))}`);
        console.error(`ConnectRPC ${method} request proto hex: ${requestBody.toString('hex')}`);
      }
      throw new Error(`ConnectRPC ${method} failed with status ${response.status()}: ${body}`);
    }
    const responseBody = Buffer.from(await response.body());
    return fromBinary(options.outputSchema, responseBody) as T;
  }

  const headers = token ? { ...CONNECT_JSON_HEADERS, Authorization: `Bearer ${token}` } : CONNECT_JSON_HEADERS;
  const response = await page.context().request.post(buildRpcUrl(servicePath, method), {
    data: payload,
    headers,
  });
  if (!response.ok()) {
    const body = await response.text();
    if (method === 'CreateAgent') {
      const formattedPayload = formatDebugPayload(payload);
      console.error(`ConnectRPC ${method} request JSON: ${formattedPayload}`);
    }
    throw new Error(`ConnectRPC ${method} failed with status ${response.status()}: ${body}`);
  }
  return (await response.json()) as T;
}

async function storeChatOrganization(page: Page, chatId: string, organizationId: string): Promise<void> {
  await page.evaluate(
    ({ key, version, chatId: chatIdValue, organizationId: organizationIdValue }) => {
      const storage = window.localStorage;
      const raw = storage.getItem(key);
      let map: Record<string, string> = {};
      if (raw) {
        try {
          const parsed = JSON.parse(raw) as { version?: number; map?: Record<string, string> } | null;
          if (parsed && parsed.version === version && parsed.map && typeof parsed.map === 'object') {
            map = { ...parsed.map };
          }
        } catch (_error) {
          map = {};
        }
      }
      map[chatIdValue] = organizationIdValue;
      storage.setItem(key, JSON.stringify({ version, map }));
    },
    {
      key: CHAT_ORG_STORAGE_KEY,
      version: CHAT_ORG_STORAGE_VERSION,
      chatId,
      organizationId,
    },
  );
}

export async function resolveIdentityId(page: Page): Promise<string> {
  const response = await postConnect<GetMeResponseWire>(page, USERS_GATEWAY_PATH, 'GetMe', {});
  const identityId = response.user?.meta?.id ?? '';
  if (!identityId) {
    throw new Error('UsersGateway.GetMe response missing identity id.');
  }
  return identityId;
}

export async function resolveUserLabel(page: Page, identityId: string): Promise<string> {
  const response = await postConnect<BatchGetUsersResponseWire>(
    page,
    USERS_GATEWAY_PATH,
    'BatchGetUsers',
    { identityIds: [identityId] },
  );
  const users = response.users ?? [];
  const match = users.find((user) => user.meta?.id === identityId);
  if (!match) {
    throw new Error(`BatchGetUsers response missing user ${identityId}.`);
  }
  const name = typeof match.name === 'string' ? match.name.trim() : '';
  const email = typeof match.email === 'string' ? match.email.trim() : '';
  if (name) return name;
  if (email) return email;
  throw new Error(`User ${identityId} missing name and email.`);
}

export async function createChat(
  page: Page,
  organizationId: string,
  participantId: string,
): Promise<string> {
  const response = await postConnect<CreateChatResponseWire>(page, CHAT_GATEWAY_PATH, 'CreateChat', {
    organizationId,
    participantIds: [participantId],
  });
  if (!response.chat?.id) {
    throw new Error('CreateChat response missing chat id.');
  }
  const chatId = response.chat.id;
  await storeChatOrganization(page, chatId, organizationId);
  return chatId;
}

export async function updateChatStatus(
  page: Page,
  chatId: string,
  status: 'open' | 'closed',
): Promise<void> {
  await postConnect(page, CHAT_GATEWAY_PATH, 'UpdateChat', {
    chatId,
    status: CHAT_STATUS_MAP[status],
  });
}

export async function createOrganization(page: Page, name: string): Promise<string> {
  const response = await postConnect<CreateOrganizationResponseWire>(
    page,
    ORGS_GATEWAY_PATH,
    'CreateOrganization',
    { name },
  );
  if (!response.organization?.id) {
    throw new Error('CreateOrganization response missing organization id.');
  }
  return response.organization.id;
}

export async function createMembership(
  page: Page,
  organizationId: string,
  identityId: string,
  role: MembershipRoleValue = 'MEMBERSHIP_ROLE_MEMBER',
): Promise<string> {
  const response = await postConnect<CreateMembershipResponseWire>(
    page,
    ORGS_GATEWAY_PATH,
    'CreateMembership',
    { organizationId, identityId, role: MEMBERSHIP_ROLE_MAP[role] },
  );
  if (!response.membership?.id) {
    throw new Error('CreateMembership response missing membership id.');
  }
  return response.membership.id;
}

export async function acceptMembership(page: Page, membershipId: string): Promise<void> {
  await postConnect(page, ORGS_GATEWAY_PATH, 'AcceptMembership', { membershipId });
}

export async function listAccessibleOrganizations(
  page: Page,
): Promise<Array<{ id: string; name: string }>> {
  const response = await postConnect<ListAccessibleOrganizationsResponseWire>(
    page,
    ORGS_GATEWAY_PATH,
    'ListAccessibleOrganizations',
    {},
  );
  return response.organizations ?? [];
}

export async function createLLMProvider(
  page: Page,
  opts: {
    endpoint: string;
    authMethod: string;
    token: string;
    organizationId: string;
    protocol?: string;
  },
): Promise<string> {
  const payload: Record<string, unknown> = {
    endpoint: opts.endpoint,
    authMethod: opts.authMethod,
    token: opts.token,
    organizationId: opts.organizationId,
  };
  if (opts.protocol) {
    payload.protocol = opts.protocol;
  }
  const response = await postConnect<CreateLLMProviderResponseWire>(
    page,
    LLM_GATEWAY_PATH,
    'CreateLLMProvider',
    payload,
  );
  if (!response.provider?.meta?.id) {
    throw new Error('CreateLLMProvider response missing provider id.');
  }
  return response.provider.meta.id;
}

export async function createModel(
  page: Page,
  opts: { name: string; llmProviderId: string; remoteName: string; organizationId: string },
): Promise<string> {
  const response = await postConnect<CreateModelResponseWire>(
    page,
    LLM_GATEWAY_PATH,
    'CreateModel',
    {
      name: opts.name,
      llmProviderId: opts.llmProviderId,
      remoteName: opts.remoteName,
      organizationId: opts.organizationId,
    },
  );
  if (!response.model?.meta?.id) {
    throw new Error('CreateModel response missing model id.');
  }
  return response.model.meta.id;
}

type CreateAgentOptions = {
  organizationId: string;
  name: string;
  role: string;
  model: string;
  description: string;
  configuration: string;
  image: string;
  initImage: string;
  availability?: AgentAvailability;
};

type CreateAgentPayload = Omit<CreateAgentOptions, 'availability'> & {
  availability: AgentAvailability.INTERNAL | AgentAvailability.PRIVATE;
};

type SetupTestAgentOptions = {
  endpoint: string;
  initImage?: string;
  protocol?: string;
  remoteName?: string;
  token?: string;
  authMethod?: string;
};

type CreateTestModelOptions = {
  organizationId: string;
  endpoint: string;
  namePrefix?: string;
  remoteName?: string;
  authMethod?: string;
  token?: string;
  protocol?: string;
};

async function waitForAgent(page: Page, organizationId: string, agentId: string): Promise<void> {
  const timeoutMs = 20000;
  const start = Date.now();
  while (Date.now() - start < timeoutMs) {
    const agents = await listAgents(page, organizationId);
    if (agents.some((agent) => agent.meta?.id === agentId)) {
      return;
    }
    await page.waitForTimeout(500);
  }
  throw new Error(`Agent ${agentId} did not appear in time.`);
}

export async function createAgent(page: Page, opts: CreateAgentOptions): Promise<string> {
  const { initImage, ...rest } = opts;
  const trimmedInitImage = initImage.trim();
  if (!trimmedInitImage) {
    throw new Error('initImage is required to create chat agents.');
  }
  const payload = buildCreateAgentPayload({ ...rest, initImage: trimmedInitImage });
  const request = create(CreateAgentRequestSchema, payload);
  if (DEBUG_CREATE_AGENT_PAYLOAD) {
    const requestBody = Buffer.from(buildCreateAgentRequestBytes(payload));
    console.info(`ConnectRPC CreateAgent request JSON: ${formatDebugPayload(toJson(CreateAgentRequestSchema, request))}`);
    console.info(`ConnectRPC CreateAgent request proto hex: ${requestBody.toString('hex')}`);
  }
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
  if (!response.agent?.meta?.id) {
    throw new Error('CreateAgent response missing agent id.');
  }
  const agentId = response.agent.meta.id;
  await waitForAgent(page, opts.organizationId, agentId);
  return agentId;
}

export function buildCreateAgentPayload(opts: CreateAgentOptions): CreateAgentPayload {
  const { availability = AgentAvailability.INTERNAL, ...rest } = opts;
  if (availability !== AgentAvailability.INTERNAL && availability !== AgentAvailability.PRIVATE) {
    throw new Error(`Unsupported agent availability: ${availability}`);
  }
  return { ...rest, availability };
}

export function buildCreateAgentRequestJson(opts: CreateAgentOptions): unknown {
  return toJson(CreateAgentRequestSchema, create(CreateAgentRequestSchema, buildCreateAgentPayload(opts)));
}

export function buildCreateAgentRequestBytes(opts: CreateAgentOptions): Uint8Array {
  return toBinary(CreateAgentRequestSchema, create(CreateAgentRequestSchema, buildCreateAgentPayload(opts)));
}

export async function createTestModel(
  page: Page,
  opts: CreateTestModelOptions,
): Promise<{ modelId: string; providerId: string; modelName: string }> {
  const now = Date.now();
  const providerId = await createLLMProvider(page, {
    endpoint: opts.endpoint,
    authMethod: opts.authMethod ?? 'AUTH_METHOD_BEARER',
    token: opts.token ?? 'test-token',
    organizationId: opts.organizationId,
    protocol: opts.protocol ?? 'PROTOCOL_RESPONSES',
  });

  const modelName = `${opts.namePrefix ?? 'e2e-model'}-${now}`;
  const modelId = await createModel(page, {
    name: modelName,
    llmProviderId: providerId,
    remoteName: opts.remoteName ?? 'simple-hello',
    organizationId: opts.organizationId,
  });

  return { modelId, providerId, modelName };
}

export async function setupTestAgent(
  page: Page,
  opts: SetupTestAgentOptions,
): Promise<{ organizationId: string; agentId: string; agentName: string; participantId: string }> {
  const now = Date.now();
  const organizationId = await createOrganization(page, `e2e-org-llm-${now}`);
  const initImage = resolveCodexInitImage(opts.initImage);

  const { modelId } = await createTestModel(page, {
    organizationId,
    endpoint: opts.endpoint,
    namePrefix: 'e2e-model',
    protocol: opts.protocol,
    remoteName: opts.remoteName,
    token: opts.token,
    authMethod: opts.authMethod,
  });

  const agentName = `e2e-codex-agent-${now}`;
  const agentId = await createAgent(page, {
    organizationId,
    name: agentName,
    role: 'You are a helpful assistant.',
    model: modelId,
    description: 'E2E test agent using TestLLM simple-hello',
    configuration: '{}',
    image: DEFAULT_TEST_AGENT_IMAGE,
    initImage,
  });
  const participantId = agentId;

  return { organizationId, agentId, agentName, participantId };
}

export async function createAgentEnv(
  page: Page,
  agentId: string,
  name: string,
  value: string,
): Promise<string> {
  const response = await postConnect<CreateEnvResponseWire>(page, AGENTS_GATEWAY_PATH, 'CreateEnv', {
    agentId,
    name,
    value,
    description: `e2e env: ${name}`,
  });
  if (!response.env?.meta?.id) {
    throw new Error(`CreateEnv response missing env id for ${name}.`);
  }
  return response.env.meta.id;
}

export async function listAgents(
  page: Page,
  organizationId: string,
): Promise<Array<{ meta?: { id?: string }; name?: string }>> {
  const agents: Array<{ meta?: { id?: string }; name?: string }> = [];
  let pageToken: string | undefined;
  let previousToken: string | undefined;
  do {
    const response = await postConnect<ListAgentsResponseWire>(page, AGENTS_GATEWAY_PATH, 'ListAgents', {
      organizationId,
      pageSize: 200,
      pageToken,
    });
    agents.push(...(response.agents ?? []));
    previousToken = pageToken;
    pageToken = response.nextPageToken;
  } while (pageToken && pageToken !== previousToken);
  return agents;
}

export async function listChats(page: Page, organizationId: string): Promise<ChatSummary[]> {
  const chats: ChatSummary[] = [];
  let pageToken: string | undefined;
  let previousToken: string | undefined;
  do {
    const response = await postConnect<GetChatsResponseWire>(page, CHAT_GATEWAY_PATH, 'GetChats', {
      organizationId,
      pageSize: 200,
      pageToken,
    });
    chats.push(...(response.chats ?? []));
    previousToken = pageToken;
    pageToken = response.nextPageToken;
  } while (pageToken && pageToken !== previousToken);
  return chats;
}

export async function waitForChatInList(
  page: Page,
  organizationId: string,
  chatId: string,
  timeoutMs = 30000,
): Promise<ChatSummary> {
  const deadline = Date.now() + timeoutMs;
  while (Date.now() < deadline) {
    const chats = await listChats(page, organizationId);
    const chat = chats.find((item) => item.id === chatId);
    if (chat) return chat;
    await new Promise((resolve) => setTimeout(resolve, 1000));
  }
  throw new Error(`Chat ${chatId} did not appear in organization ${organizationId} list within ${timeoutMs}ms`);
}

export async function getMessages(page: Page, chatId: string): Promise<Message[]> {
  const response = await postConnect<GetMessagesResponseWire>(page, CHAT_GATEWAY_PATH, 'GetMessages', {
    chatId,
  });
  return response.messages ?? [];
}

export async function waitForAgentReply(
  page: Page,
  chatId: string,
  senderIdToExclude: string,
  timeoutMs = 120000,
  intervalMs = 3000,
): Promise<Message> {
  const deadline = Date.now() + timeoutMs;
  while (Date.now() < deadline) {
    const messages = await getMessages(page, chatId);
    const agentMsg = messages.find((message) => {
      if (!message.body) return false;
      if (message.senderId === senderIdToExclude) return false;
      return true;
    });
    if (agentMsg) return agentMsg;
    await new Promise((resolve) => setTimeout(resolve, intervalMs));
  }
  throw new Error(`Agent did not reply within ${timeoutMs}ms`);
}

export async function sendChatMessage(
  page: Page,
  chatId: string,
  message: string,
): Promise<void> {
  await postConnect(page, CHAT_GATEWAY_PATH, 'SendMessage', {
    chatId,
    body: message,
  });
}
