import { createClient } from '@connectrpc/connect';
import { createGrpcTransport } from '@connectrpc/connect-node';
import { create } from '@bufbuild/protobuf';
import { TimestampSchema, type Timestamp } from '@bufbuild/protobuf/wkt';
import { randomUUID } from 'crypto';
import type { Page } from '@playwright/test';
import { MeteringService, UsageRecordSchema } from '../../src/gen/agynio/api/metering/v1/metering_pb';
import type { Unit } from '../../src/gen/agynio/api/metering/v1/metering_pb';
import { readOidcSession } from './oidc-helpers';

const USERS_GATEWAY_PATH = '/api/agynio.api.gateway.v1.UsersGateway';
const ORGS_GATEWAY_PATH = '/api/agynio.api.gateway.v1.OrganizationsGateway';
const SECRETS_GATEWAY_PATH = '/api/agynio.api.gateway.v1.SecretsGateway';
const AGENTS_GATEWAY_PATH = '/api/agynio.api.gateway.v1.AgentsGateway';
const LLM_GATEWAY_PATH = '/api/agynio.api.gateway.v1.LLMGateway';
const RUNNERS_GATEWAY_PATH = '/api/agynio.api.gateway.v1.RunnersGateway';
const THREADS_GATEWAY_PATH = '/api/agynio.api.gateway.v1.ThreadsGateway';
const METERING_GATEWAY_PATH = '/api/agynio.api.gateway.v1.MeteringGateway';

const CONNECT_HEADERS = {
  'Content-Type': 'application/json',
  'Connect-Protocol-Version': '1',
};

const METERING_GRPC_URL = process.env.METERING_GRPC_URL ?? 'http://metering:50051';
const meteringTransport = createGrpcTransport({ baseUrl: METERING_GRPC_URL });
const meteringClient = createClient(MeteringService, meteringTransport);

type OrganizationWire = {
  id: string;
  name: string;
};

type UserWire = {
  meta?: { id?: string };
  email?: string;
  name?: string;
  nickname?: string;
};

type MembershipWire = {
  id?: string;
  identityId?: string;
  role?: string | number;
  status?: string | number;
};

type RunnerWire = {
  meta?: { id?: string };
  name?: string;
  labels?: Record<string, string>;
  status?: string | number;
};

type DeviceWire = {
  meta?: { id?: string; createdAt?: string };
  name?: string;
  status?: string | number;
  enrollmentJwt?: string;
};

type ThreadParticipantWire = {
  id?: string;
  joinedAt?: string;
  passive?: boolean;
};

type ThreadWire = {
  id?: string;
  participants?: ThreadParticipantWire[];
  status?: string | number;
  createdAt?: string;
  updatedAt?: string;
  organizationId?: string;
  messageCount?: number;
};

type MessageWire = {
  id?: string;
  threadId?: string;
  senderId?: string;
  body?: string;
  fileIds?: string[];
  createdAt?: string;
};

type CreateOrganizationResponseWire = {
  organization?: { id?: string };
};

type CreateThreadResponseWire = {
  thread?: ThreadWire;
};

type SendMessageResponseWire = {
  message?: MessageWire;
};

type ListAccessibleOrganizationsResponseWire = {
  organizations?: OrganizationWire[];
};

type CreateUserResponseWire = {
  user?: { meta?: { id?: string } };
};

type UpdateUserResponseWire = {
  user?: { meta?: { id?: string } };
};

type ListUsersResponseWire = {
  users?: UserWire[];
};

type GetMeResponseWire = {
  user?: UserWire;
  clusterRole?: string | number;
};

type CreateMembershipResponseWire = {
  membership?: MembershipWire;
};

type ListMembersResponseWire = {
  memberships?: MembershipWire[];
};

type UpdateMembershipRoleResponseWire = {
  membership?: MembershipWire;
};

type ClusterRoleWire = string | number | undefined;

type CreateSecretProviderResponseWire = {
  secretProvider?: { meta?: { id?: string } };
};

type CreateLLMProviderResponseWire = {
  provider?: { meta?: { id?: string } };
};

type ListSecretProvidersResponseWire = {
  secretProviders?: Array<{ meta?: { id?: string }; title?: string }>;
};

type SecretWire = {
  meta?: { id?: string };
  secretProviderId?: string;
};

type AgentWire = {
  meta?: { id?: string };
  name?: string;
  nickname?: string;
  role?: string;
  model?: string;
  description?: string;
  configuration?: string;
  image?: string;
  initImage?: string;
  organizationId?: string;
};

type McpWire = {
  meta?: { id?: string };
  name?: string;
  agentId?: string;
};

type HookWire = {
  meta?: { id?: string };
  event?: string;
  agentId?: string;
};

type ModelWire = {
  meta?: { id?: string };
  id?: string;
  name?: string;
  llmProviderId?: string;
  remoteName?: string;
};

type LlmProviderWire = {
  meta?: { id?: string };
  id?: string;
  endpoint?: string;
  authMethod?: string | number;
  organizationId?: string;
};

type CreateSecretResponseWire = {
  secret?: { meta?: { id?: string } };
};
type CreateImagePullSecretResponseWire = {
  imagePullSecret?: { meta?: { id?: string } };
};

type ListSecretsResponseWire = {
  secrets?: SecretWire[];
};

type CreateDeviceResponseWire = {
  device?: DeviceWire;
  enrollmentJwt?: string;
};

type ListDevicesResponseWire = {
  devices?: DeviceWire[];
};

type ListModelsResponseWire = {
  models?: ModelWire[];
};

type ListLlmProvidersResponseWire = {
  providers?: LlmProviderWire[];
};

type CreateLlmProviderResponseWire = {
  provider?: LlmProviderWire;
};

type CreateModelResponseWire = {
  model?: ModelWire;
};

type EntityWithId = { meta?: { id?: string }; id?: string };

type CreateAgentResponseWire = {
  agent?: AgentWire;
};

type CreateMcpResponseWire = {
  mcp?: McpWire;
};

type CreateHookResponseWire = {
  hook?: HookWire;
};

type ListRunnersResponseWire = {
  runners?: RunnerWire[];
};

type RegisterRunnerResponseWire = {
  runner?: RunnerWire;
  serviceToken?: string;
};

type GetRunnerResponseWire = {
  runner?: RunnerWire;
};

type UsageBucketWire = {
  value?: string | number;
};

type QueryUsageResponseWire = {
  buckets?: UsageBucketWire[];
};

type UsageRecordInput = {
  labels: Record<string, string>;
  unit: Unit;
  value: bigint;
  timestamp?: Date;
  producer?: string;
  idempotencyKey?: string;
};

type MembershipRoleValue = 'MEMBERSHIP_ROLE_UNSPECIFIED' | 'MEMBERSHIP_ROLE_OWNER' | 'MEMBERSHIP_ROLE_MEMBER';
type MembershipStatusValue =
  | 'MEMBERSHIP_STATUS_UNSPECIFIED'
  | 'MEMBERSHIP_STATUS_PENDING'
  | 'MEMBERSHIP_STATUS_ACTIVE';

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

function toProtoTimestamp(date: Date): Timestamp {
  return create(TimestampSchema, {
    seconds: BigInt(Math.floor(date.getTime() / 1000)),
    nanos: 0,
  });
}

function findEntityId(entities: EntityWithId[] | undefined): string {
  if (!entities) return '';
  for (const entity of entities) {
    const id = entity.meta?.id ?? entity.id ?? '';
    if (id) return id;
  }
  return '';
}

const CLUSTER_ROLE_ADMIN = 1;

function isClusterAdminRole(role: ClusterRoleWire): boolean {
  if (role === undefined || role === null) return false;
  if (typeof role === 'number') {
    return role === CLUSTER_ROLE_ADMIN;
  }
  return role === 'CLUSTER_ROLE_ADMIN';
}

async function postConnect<T>(
  page: Page,
  servicePath: string,
  method: string,
  payload: Record<string, unknown>,
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

async function postConnectAsClusterAdmin<T>(
  page: Page,
  servicePath: string,
  method: string,
  payload: Record<string, unknown>,
): Promise<T> {
  const token = process.env.CLUSTER_ADMIN_TOKEN;
  if (!token) {
    throw new Error('CLUSTER_ADMIN_TOKEN env var is required for admin operations in e2e.');
  }
  const headers = { ...CONNECT_HEADERS, Authorization: `Bearer ${token}` };
  const response = await page.context().request.post(buildRpcUrl(servicePath, method), {
    data: payload,
    headers,
  });
  if (!response.ok()) {
    const body = await response.text();
    throw new Error(`ConnectRPC ${method} (admin) failed with status ${response.status()}: ${body}`);
  }
  return (await response.json()) as T;
}

function isNotFoundError(error: unknown): boolean {
  if (!(error instanceof Error)) return false;
  return error.message.includes('status 404');
}

function isAlreadyClusterAdminError(error: unknown): boolean {
  if (!(error instanceof Error)) return false;
  return (
    error.message.includes('tuple to be written already existed') ||
    error.message.includes('tuple to be written already exists')
  );
}

export async function getMe(page: Page): Promise<GetMeResponseWire> {
  const response = await postConnect<GetMeResponseWire>(page, USERS_GATEWAY_PATH, 'GetMe', {});
  if (!response.user?.meta?.id) {
    throw new Error('GetMe response missing user identity id.');
  }
  return response;
}

export async function ensureClusterAdmin(page: Page): Promise<void> {
  const me = await getMe(page);
  if (isClusterAdminRole(me.clusterRole)) {
    return;
  }
  const identityId = me.user?.meta?.id;
  if (!identityId) {
    throw new Error('GetMe response missing identity id for cluster role update.');
  }
  try {
    await postConnectAsClusterAdmin<UpdateUserResponseWire>(page, USERS_GATEWAY_PATH, 'UpdateUser', {
      identityId,
      clusterRole: 'CLUSTER_ROLE_ADMIN',
    });
  } catch (error) {
    if (!isAlreadyClusterAdminError(error)) {
      throw error;
    }
  }
  const start = Date.now();
  while (Date.now() - start < 10000) {
    const updated = await getMe(page);
    if (isClusterAdminRole(updated.clusterRole)) {
      return;
    }
    await page.waitForTimeout(500);
  }
  throw new Error('Cluster role update did not propagate in time.');
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

export async function listAccessibleOrganizations(page: Page): Promise<OrganizationWire[]> {
  const me = await getMe(page);
  const identityId = me.user?.meta?.id;
  if (!identityId) {
    throw new Error('GetMe response missing identity id for ListAccessibleOrganizations.');
  }
  const response = await postConnect<ListAccessibleOrganizationsResponseWire>(
    page,
    ORGS_GATEWAY_PATH,
    'ListAccessibleOrganizations',
    { identityId },
  );
  return response.organizations ?? [];
}

async function waitForOrganization(page: Page, organizationId: string): Promise<void> {
  const timeoutMs = 10000;
  const start = Date.now();
  while (Date.now() - start < timeoutMs) {
    const organizations = await listAccessibleOrganizations(page);
    if (organizations.some((org) => org.id === organizationId)) {
      return;
    }
    await page.waitForTimeout(500);
  }
  throw new Error(`Organization ${organizationId} did not appear in time.`);
}

export async function setSelectedOrganization(page: Page, organizationId: string): Promise<void> {
  await waitForOrganization(page, organizationId);
  await page.evaluate((orgId) => {
    window.localStorage.setItem(
      'console.contextMode',
      JSON.stringify({ mode: 'organization', organizationId: orgId }),
    );
    window.localStorage.removeItem('console.selectedOrganization');
  }, organizationId);
}

export async function createThread(
  page: Page,
  opts: { organizationId: string; participantIds: string[] },
): Promise<string> {
  const me = await getMe(page);
  const initiatorId = me.user?.meta?.id ?? '';
  const participantIds = opts.participantIds.filter((participantId) => participantId !== initiatorId);
  const response = await postConnect<CreateThreadResponseWire>(
    page,
    THREADS_GATEWAY_PATH,
    'CreateThread',
    {
      organizationId: opts.organizationId,
      participantIds,
    },
  );
  const threadId = response.thread?.id;
  if (!threadId) {
    throw new Error('CreateThread response missing thread id.');
  }
  return threadId;
}

export async function sendThreadMessage(
  page: Page,
  opts: { threadId: string; senderId: string; body: string; fileIds?: string[] },
): Promise<string> {
  const response = await postConnect<SendMessageResponseWire>(
    page,
    THREADS_GATEWAY_PATH,
    'SendMessage',
    {
      threadId: opts.threadId,
      senderId: opts.senderId,
      body: opts.body,
      fileIds: opts.fileIds ?? [],
    },
  );
  const messageId = response.message?.id;
  if (!messageId) {
    throw new Error('SendMessage response missing message id.');
  }
  return messageId;
}

export async function createUser(
  page: Page,
  opts: { email: string; nickname: string },
): Promise<string> {
  const now = Date.now();
  const oidcSubject = `e2e-${opts.email}-${now}`;
  const response = await postConnect<CreateUserResponseWire>(page, USERS_GATEWAY_PATH, 'CreateUser', {
    oidcSubject,
    name: opts.email,
    nickname: opts.nickname,
    clusterRole: 'CLUSTER_ROLE_UNSPECIFIED',
  });
  const identityId = response.user?.meta?.id;
  if (!identityId) {
    throw new Error('CreateUser response missing identity id.');
  }
  await postConnect<UpdateUserResponseWire>(page, USERS_GATEWAY_PATH, 'UpdateUser', {
    identityId,
    email: opts.email,
    nickname: opts.nickname,
  });
  return identityId;
}

export async function listUsers(page: Page): Promise<UserWire[]> {
  const response = await postConnect<ListUsersResponseWire>(page, USERS_GATEWAY_PATH, 'ListUsers', {
    pageSize: 200,
    pageToken: '',
  });
  return response.users ?? [];
}

async function resolveIdentityIdByEmail(page: Page, email: string): Promise<string> {
  const users = await listUsers(page);
  const match = users.find((user) => user.email === email);
  if (match?.meta?.id) {
    return match.meta.id;
  }
  return createUser(page, { email, nickname: email.split('@')[0] ?? 'e2e-user' });
}

export async function inviteMember(
  page: Page,
  opts: { organizationId: string; email: string; role: MembershipRoleValue },
): Promise<{ identityId: string; membershipId: string }> {
  const identityId = await resolveIdentityIdByEmail(page, opts.email);
  const response = await postConnect<CreateMembershipResponseWire>(
    page,
    ORGS_GATEWAY_PATH,
    'CreateMembership',
    {
      organizationId: opts.organizationId,
      identityId,
      role: opts.role,
    },
  );
  const membershipId = response.membership?.id;
  if (!membershipId) {
    throw new Error('CreateMembership response missing membership id.');
  }
  return { identityId, membershipId };
}

export async function listMembers(
  page: Page,
  opts: { organizationId: string; status?: MembershipStatusValue },
): Promise<MembershipWire[]> {
  const response = await postConnect<ListMembersResponseWire>(
    page,
    ORGS_GATEWAY_PATH,
    'ListMembers',
    {
      organizationId: opts.organizationId,
      status: opts.status ?? 'MEMBERSHIP_STATUS_UNSPECIFIED',
      pageSize: 200,
      pageToken: '',
    },
  );
  return response.memberships ?? [];
}

async function resolveMembership(
  page: Page,
  organizationId: string,
  identityId: string,
): Promise<MembershipWire> {
  const memberships = await listMembers(page, {
    organizationId,
    status: 'MEMBERSHIP_STATUS_UNSPECIFIED',
  });
  const membership = memberships.find((member) => member.identityId === identityId);
  if (!membership?.id) {
    throw new Error(`Membership not found for identity ${identityId}.`);
  }
  return membership;
}

export async function removeMember(
  page: Page,
  opts: { organizationId: string; identityId: string },
): Promise<void> {
  const membership = await resolveMembership(page, opts.organizationId, opts.identityId);
  await postConnect(page, ORGS_GATEWAY_PATH, 'RemoveMembership', {
    membershipId: membership.id,
  });
}

export async function changeMemberRole(
  page: Page,
  opts: { organizationId: string; identityId: string; role: MembershipRoleValue },
): Promise<void> {
  const membership = await resolveMembership(page, opts.organizationId, opts.identityId);
  const response = await postConnect<UpdateMembershipRoleResponseWire>(
    page,
    ORGS_GATEWAY_PATH,
    'UpdateMembershipRole',
    {
      membershipId: membership.id,
      role: opts.role,
    },
  );
  if (!response.membership?.id) {
    throw new Error('UpdateMembershipRole response missing membership id.');
  }
}

export async function createSecretProvider(
  page: Page,
  opts: { organizationId: string; name: string; url: string },
): Promise<string> {
  const response = await postConnect<CreateSecretProviderResponseWire>(
    page,
    SECRETS_GATEWAY_PATH,
    'CreateSecretProvider',
    {
      title: opts.name,
      description: `E2E provider for ${opts.name}`,
      type: 'SECRET_PROVIDER_TYPE_VAULT',
      config: {
        vault: {
          address: opts.url,
          token: 'e2e-token',
        },
      },
      organizationId: opts.organizationId,
    },
  );
  const providerId = response.secretProvider?.meta?.id;
  if (!providerId) {
    throw new Error('CreateSecretProvider response missing provider id.');
  }
  return providerId;
}

export async function listSecretProviders(
  page: Page,
  opts: { organizationId: string },
): Promise<Array<{ meta?: { id?: string }; title?: string }>> {
  const response = await postConnect<ListSecretProvidersResponseWire>(
    page,
    SECRETS_GATEWAY_PATH,
    'ListSecretProviders',
    {
      organizationId: opts.organizationId,
      pageSize: 200,
      pageToken: '',
    },
  );
  return response.secretProviders ?? [];
}

export async function listSecrets(
  page: Page,
  opts: { organizationId: string; providerId?: string },
): Promise<SecretWire[]> {
  const response = await postConnect<ListSecretsResponseWire>(page, SECRETS_GATEWAY_PATH, 'ListSecrets', {
    organizationId: opts.organizationId,
    pageSize: 200,
    pageToken: '',
    secretProviderId: opts.providerId ?? '',
  });
  return response.secrets ?? [];
}

export async function listLlmProviders(
  page: Page,
  opts: { organizationId: string },
): Promise<LlmProviderWire[]> {
  const response = await postConnect<ListLlmProvidersResponseWire>(page, LLM_GATEWAY_PATH, 'ListLLMProviders', {
    organizationId: opts.organizationId,
    pageSize: 200,
    pageToken: '',
  });
  return response.providers ?? [];
}

export async function listModels(
  page: Page,
  opts: { organizationId: string; llmProviderId?: string },
): Promise<ModelWire[]> {
  const response = await postConnect<ListModelsResponseWire>(page, LLM_GATEWAY_PATH, 'ListModels', {
    organizationId: opts.organizationId,
    llmProviderId: opts.llmProviderId ?? '',
    pageSize: 200,
    pageToken: '',
  });
  return response.models ?? [];
}

async function createLlmProvider(
  page: Page,
  opts: { organizationId: string; endpoint: string; token: string; authMethod?: string | number },
): Promise<string> {
  const response = await postConnect<CreateLlmProviderResponseWire>(page, LLM_GATEWAY_PATH, 'CreateLLMProvider', {
    endpoint: opts.endpoint,
    authMethod: opts.authMethod ?? 'AUTH_METHOD_BEARER',
    token: opts.token,
    organizationId: opts.organizationId,
  });
  const providerId = response.provider?.meta?.id ?? response.provider?.id ?? '';
  if (!providerId) {
    throw new Error('CreateLLMProvider response missing provider id.');
  }
  return providerId;
}

async function createModelInternal(
  page: Page,
  opts: { organizationId: string; llmProviderId: string; name: string; remoteName: string },
): Promise<string> {
  const response = await postConnect<CreateModelResponseWire>(page, LLM_GATEWAY_PATH, 'CreateModel', {
    name: opts.name,
    llmProviderId: opts.llmProviderId,
    remoteName: opts.remoteName,
    organizationId: opts.organizationId,
  });
  const modelId = response.model?.meta?.id ?? response.model?.id ?? '';
  if (!modelId) {
    throw new Error('CreateModel response missing model id.');
  }
  return modelId;
}

async function ensureLlmProviderId(page: Page, organizationId: string): Promise<string> {
  const providers = await listLlmProviders(page, { organizationId });
  const providerId = findEntityId(providers);
  if (providerId) return providerId;

  const now = Date.now();
  return createLlmProvider(page, {
    organizationId,
    endpoint: `https://llm.e2e.agyn.dev/${now}`,
    token: `e2e-token-${now}`,
  });
}

async function ensureModelId(page: Page, organizationId: string): Promise<string> {
  const models = await listModels(page, { organizationId });
  const modelId = findEntityId(models);
  if (modelId) return modelId;

  const providerId = await ensureLlmProviderId(page, organizationId);
  const now = Date.now();
  return createModelInternal(page, {
    organizationId,
    llmProviderId: providerId,
    name: `E2E Model ${now}`,
    remoteName: 'gpt-4o-mini',
  });
}

export async function createSecret(
  page: Page,
  opts: { providerId: string; name: string; value: string; organizationId: string },
): Promise<string> {
  const response = await postConnect<CreateSecretResponseWire>(page, SECRETS_GATEWAY_PATH, 'CreateSecret', {
    title: opts.name,
    description: `E2E secret for ${opts.name}`,
    secretProviderId: opts.providerId,
    remoteName: opts.value,
    organizationId: opts.organizationId,
  });
  const secretId = response.secret?.meta?.id;
  if (!secretId) {
    throw new Error('CreateSecret response missing secret id.');
  }
  return secretId;
}

export async function createImagePullSecret(
  page: Page,
  opts: { organizationId: string; registry: string; username: string; value: string; description?: string },
): Promise<string> {
  const response = await postConnect<CreateImagePullSecretResponseWire>(
    page,
    SECRETS_GATEWAY_PATH,
    'CreateImagePullSecret',
    {
      description: opts.description ?? `E2E image pull secret for ${opts.registry}`,
      registry: opts.registry,
      username: opts.username,
      value: opts.value,
      organizationId: opts.organizationId,
    },
  );
  const secretId = response.imagePullSecret?.meta?.id;
  if (!secretId) {
    throw new Error('CreateImagePullSecret response missing image pull secret id.');
  }
  return secretId;
}

export async function createAgent(
  page: Page,
  opts: {
    organizationId: string;
    name: string;
    nickname?: string;
    role?: string;
    model?: string;
    description?: string;
    configuration?: string;
    image?: string;
    initImage?: string;
  },
): Promise<string> {
  let modelId = opts.model?.trim() ?? '';
  if (!modelId) {
    modelId = await ensureModelId(page, opts.organizationId);
  }
  const payload: {
    name: string;
    nickname?: string;
    role: string;
    model: string;
    description: string;
    configuration: string;
    image: string;
    initImage: string;
    organizationId: string;
  } = {
    name: opts.name,
    role: opts.role ?? 'assistant',
    model: modelId,
    description: opts.description ?? '',
    configuration: opts.configuration ?? '',
    image: opts.image ?? '',
    initImage: opts.initImage ?? '',
    organizationId: opts.organizationId,
  };
  const trimmedNickname = opts.nickname?.trim();
  if (trimmedNickname) {
    payload.nickname = trimmedNickname;
  }
  const response = await postConnect<CreateAgentResponseWire>(page, AGENTS_GATEWAY_PATH, 'CreateAgent', payload);
  const agentId = response.agent?.meta?.id;
  if (!agentId) {
    throw new Error('CreateAgent response missing agent id.');
  }
  return agentId;
}

export async function createMcp(
  page: Page,
  opts: { agentId: string; name: string; image: string; command: string; description?: string },
): Promise<string> {
  const response = await postConnect<CreateMcpResponseWire>(page, AGENTS_GATEWAY_PATH, 'CreateMcp', {
    agentId: opts.agentId,
    name: opts.name,
    image: opts.image,
    command: opts.command,
    description: opts.description ?? '',
  });
  const mcpId = response.mcp?.meta?.id;
  if (!mcpId) {
    throw new Error('CreateMcp response missing mcp id.');
  }
  return mcpId;
}

export async function createHook(
  page: Page,
  opts: {
    agentId: string;
    event: string;
    functionName: string;
    image: string;
    description?: string;
  },
): Promise<string> {
  const response = await postConnect<CreateHookResponseWire>(page, AGENTS_GATEWAY_PATH, 'CreateHook', {
    agentId: opts.agentId,
    event: opts.event,
    function: opts.functionName,
    image: opts.image,
    description: opts.description ?? '',
  });
  const hookId = response.hook?.meta?.id;
  if (!hookId) {
    throw new Error('CreateHook response missing hook id.');
  }
  return hookId;
}

export async function deleteSecret(page: Page, secretId: string): Promise<void> {
  try {
    await postConnect(page, SECRETS_GATEWAY_PATH, 'DeleteSecret', { id: secretId });
  } catch (error) {
    if (isNotFoundError(error)) return;
    throw error;
  }
}

export async function deleteSecretProvider(page: Page, providerId: string): Promise<void> {
  try {
    await postConnect(page, SECRETS_GATEWAY_PATH, 'DeleteSecretProvider', { id: providerId });
  } catch (error) {
    if (isNotFoundError(error)) return;
    throw error;
  }
}

export async function clearOrganizationSecrets(page: Page, organizationId: string): Promise<void> {
  const timeoutMs = 45000;
  const start = Date.now();
  while (Date.now() - start < timeoutMs) {
    const secrets = await listSecrets(page, { organizationId });
    const providers = await listSecretProviders(page, { organizationId });
    if (secrets.length === 0 && providers.length === 0) {
      return;
    }
    await Promise.all(
      secrets
        .map((secret) => secret.meta?.id)
        .filter((secretId): secretId is string => Boolean(secretId))
        .map((secretId) => deleteSecret(page, secretId)),
    );
    await Promise.all(
      providers
        .map((provider) => provider.meta?.id)
        .filter((providerId): providerId is string => Boolean(providerId))
        .map((providerId) => deleteSecretProvider(page, providerId)),
    );
    await page.waitForTimeout(500);
  }
  throw new Error('Timed out clearing organization secrets.');
}

export async function createLLMProvider(
  page: Page,
  opts: { organizationId: string; endpoint: string; authMethod: string; token: string; protocol: string },
): Promise<string> {
  const response = await postConnect<CreateLLMProviderResponseWire>(page, LLM_GATEWAY_PATH, 'CreateLLMProvider', {
    endpoint: opts.endpoint,
    authMethod: opts.authMethod,
    token: opts.token,
    organizationId: opts.organizationId,
    protocol: opts.protocol,
  });
  const providerId = response.provider?.meta?.id;
  if (!providerId) {
    throw new Error('CreateLLMProvider response missing provider id.');
  }
  return providerId;
}

export async function createModel(
  page: Page,
  opts: { organizationId: string; providerId: string; name: string; remoteName: string },
): Promise<string> {
  const response = await postConnect<CreateModelResponseWire>(page, LLM_GATEWAY_PATH, 'CreateModel', {
    name: opts.name,
    llmProviderId: opts.providerId,
    remoteName: opts.remoteName,
    organizationId: opts.organizationId,
  });
  const modelId = response.model?.meta?.id;
  if (!modelId) {
    throw new Error('CreateModel response missing model id.');
  }
  return modelId;
}

export async function recordUsage(organizationId: string, records: UsageRecordInput[]): Promise<void> {
  if (!organizationId) {
    throw new Error('Organization id is required to record usage events.');
  }
  if (records.length === 0) {
    throw new Error('Usage records are required to record usage events.');
  }
  const now = new Date();
  const payloadRecords = records.map((record) => {
    const timestamp = record.timestamp ?? now;
    return create(UsageRecordSchema, {
      orgId: organizationId,
      idempotencyKey: record.idempotencyKey ?? randomUUID(),
      producer: record.producer ?? 'e2e',
      timestamp: toProtoTimestamp(timestamp),
      labels: record.labels,
      unit: record.unit,
      value: record.value,
    });
  });
  await meteringClient.record({ records: payloadRecords });
}

export async function queryUsage(
  page: Page,
  opts: {
    organizationId: string;
    start: string;
    end: string;
    unit: string;
    granularity: string;
    groupBy?: string;
    labelFilters?: Record<string, string>;
  },
): Promise<QueryUsageResponseWire> {
  return postConnect<QueryUsageResponseWire>(page, METERING_GATEWAY_PATH, 'QueryUsage', {
    orgId: opts.organizationId,
    start: opts.start,
    end: opts.end,
    unit: opts.unit,
    granularity: opts.granularity,
    groupBy: opts.groupBy ?? '',
    labelFilters: opts.labelFilters ?? {},
  });
}

export async function createDevice(page: Page, opts: { name: string }): Promise<CreateDeviceResponseWire> {
  const response = await postConnect<CreateDeviceResponseWire>(page, USERS_GATEWAY_PATH, 'CreateDevice', {
    name: opts.name,
  });
  if (!response.device?.meta?.id) {
    throw new Error('CreateDevice response missing device id.');
  }
  return response;
}

export async function listDevices(page: Page): Promise<DeviceWire[]> {
  const response = await postConnect<ListDevicesResponseWire>(page, USERS_GATEWAY_PATH, 'ListDevices', {
    pageSize: 200,
    pageToken: '',
  });
  return response.devices ?? [];
}

export async function deleteDevice(page: Page, deviceId: string): Promise<void> {
  try {
    await postConnect(page, USERS_GATEWAY_PATH, 'DeleteDevice', { id: deviceId });
  } catch (error) {
    if (isNotFoundError(error)) return;
    throw error;
  }
}

export async function listRunners(page: Page): Promise<RunnerWire[]> {
  const response = await postConnect<ListRunnersResponseWire>(page, RUNNERS_GATEWAY_PATH, 'ListRunners', {
    pageSize: 200,
    pageToken: '',
  });
  return response.runners ?? [];
}

export async function registerRunner(
  page: Page,
  opts: {
    name: string;
    labels?: Record<string, string>;
    organizationId?: string;
    capabilities?: string[];
  },
): Promise<RunnerWire> {
  const payload: Record<string, unknown> = {
    name: opts.name,
    labels: opts.labels ?? {},
    capabilities: opts.capabilities ?? [],
  };
  if (opts.organizationId) {
    payload.organizationId = opts.organizationId;
  }
  const response = await postConnect<RegisterRunnerResponseWire>(
    page,
    RUNNERS_GATEWAY_PATH,
    'RegisterRunner',
    payload,
  );
  if (!response.runner?.meta?.id) {
    throw new Error('RegisterRunner response missing runner id.');
  }
  return response.runner;
}

export async function getRunner(page: Page, runnerId: string): Promise<RunnerWire> {
  const response = await postConnect<GetRunnerResponseWire>(page, RUNNERS_GATEWAY_PATH, 'GetRunner', {
    id: runnerId,
  });
  if (!response.runner) {
    throw new Error('GetRunner response missing runner.');
  }
  return response.runner;
}
