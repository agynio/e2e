import { enumToJson } from '@bufbuild/protobuf';
import type { Page } from '@playwright/test';
import { ChatStatus, ChatStatusSchema } from '../../src/gen/agynio/api/chat/v1/chat_pb';
import { MembershipRole, MembershipRoleSchema } from '../../src/gen/agynio/api/organizations/v1/organizations_pb';

const CHAT_GATEWAY_PATH = '/api/agynio.api.gateway.v1.ChatGateway';
const AGENTS_GATEWAY_PATH = '/api/agynio.api.gateway.v1.AgentsGateway';
const LLM_GATEWAY_PATH = '/api/agynio.api.gateway.v1.LLMGateway';
const ORGS_GATEWAY_PATH = '/api/agynio.api.gateway.v1.OrganizationsGateway';
const USERS_GATEWAY_PATH = '/api/agynio.api.gateway.v1.UsersGateway';

export const DEFAULT_TEST_INIT_IMAGE =
  process.env.E2E_AGENT_INIT_IMAGE ??
  process.env.CODEX_INIT_IMAGE ??
  'ghcr.io/agynio/agent-init-codex:0.13.19';

export const DEFAULT_TEST_AGENT_IMAGE = 'alpine:3.21';

const CONNECT_HEADERS = {
  'Content-Type': 'application/json',
  'Connect-Protocol-Version': '1',
};

type CreateChatResponseWire = {
  chat?: { id?: string };
};

type CreateOrganizationResponseWire = {
  organization?: { id?: string };
};

type CreateMembershipResponseWire = {
  membership?: { id?: string };
};

type CreateAPITokenResponseWire = {
  plaintextToken?: string;
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

function enumName(schema: Parameters<typeof enumToJson>[0], value: number): string {
  const jsonValue = enumToJson(schema, value as never);
  if (typeof jsonValue !== 'string') {
    throw new Error(`Expected enum ${schema.typeName} to serialize as string.`);
  }
  return jsonValue;
}

const CHAT_STATUS_MAP = {
  open: enumName(ChatStatusSchema, ChatStatus.OPEN),
  closed: enumName(ChatStatusSchema, ChatStatus.CLOSED),
} satisfies Record<'open' | 'closed', string>;

const MEMBERSHIP_ROLE_MAP = {
  MEMBERSHIP_ROLE_OWNER: enumName(MembershipRoleSchema, MembershipRole.OWNER),
  MEMBERSHIP_ROLE_MEMBER: enumName(MembershipRoleSchema, MembershipRole.MEMBER),
} satisfies Record<MembershipRoleValue, string>;

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
): Promise<T> {
  const session = await readOidcSession(page);
  const token = session?.accessToken ?? null;
  const headers = token ? { ...CONNECT_HEADERS, Authorization: `Bearer ${token}` } : CONNECT_HEADERS;
  const response = await page.context().request.post(buildRpcUrl(servicePath, method), {
    data: payload,
    headers,
  });
  if (!response.ok()) {
    const body = await response.text();
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
  const session = await readOidcSession(page);
  const token = session?.accessToken ?? null;
  const headers: Record<string, string> = token
    ? { Authorization: `Bearer ${token}` }
    : {};

  const baseUrl = resolveBaseUrl();
  const response = await page.context().request.get(`${baseUrl}/api/me`, { headers });
  if (!response.ok()) {
    const body = await response.text();
    throw new Error(`GET /api/me failed with status ${response.status()}: ${body}`);
  }

  const payload = (await response.json()) as { identity_id?: string };
  if (!payload.identity_id) {
    throw new Error('/api/me response missing identity_id');
  }
  return payload.identity_id;
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

export async function createApiToken(page: Page, name: string): Promise<string> {
  const response = await postConnect<CreateAPITokenResponseWire>(
    page,
    USERS_GATEWAY_PATH,
    'CreateAPIToken',
    { name },
  );
  const token = response.plaintextToken;
  if (!token) {
    throw new Error('CreateAPIToken response missing plaintext token.');
  }
  return token;
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
};

type SetupTestAgentOptions = {
  endpoint: string;
  initImage?: string;
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
  const payload = { ...rest, initImage };
  const response = await postConnect<CreateAgentResponseWire>(
    page,
    AGENTS_GATEWAY_PATH,
    'CreateAgent',
    payload,
  );
  if (!response.agent?.meta?.id) {
    throw new Error('CreateAgent response missing agent id.');
  }
  const agentId = response.agent.meta.id;
  await waitForAgent(page, opts.organizationId, agentId);
  return agentId;
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
  const initImage = opts.initImage ?? DEFAULT_TEST_INIT_IMAGE;
  const apiToken = await createApiToken(page, `e2e-agent-token-${now}`);

  const { modelId } = await createTestModel(page, {
    organizationId,
    endpoint: opts.endpoint,
    namePrefix: 'e2e-model',
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
  await createAgentEnv(page, agentId, 'LLM_API_TOKEN', apiToken);
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
  const initialMessages = await getMessages(page, chatId);
  const seenMessageIds = new Set(
    initialMessages
      .map((message) => message.id)
      .filter((id): id is string => typeof id === 'string' && id.length > 0),
  );
  const deadline = Date.now() + timeoutMs;
  while (Date.now() < deadline) {
    const messages = await getMessages(page, chatId);
    const agentMsg = messages.find((message) => {
      if (!message.body) return false;
      if (message.senderId === senderIdToExclude) return false;
      if (message.id && seenMessageIds.has(message.id)) return false;
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
