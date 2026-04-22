import http from 'node:http';
import http2 from 'node:http2';
import { randomUUID } from 'node:crypto';
import { BinaryReader, WireType } from '@bufbuild/protobuf/wire';

const port = Number(process.env.MOCK_SERVER_PORT ?? 5000);
const meteringPort = Number(process.env.MOCK_METERING_PORT ?? 50051);
const proxyTarget = new URL(process.env.MOCK_PROXY_TARGET ?? 'http://127.0.0.1:4173');
const oidcAuthority = process.env.MOCK_OIDC_AUTHORITY ?? `http://127.0.0.1:${port}`;
const oidcClientId = process.env.MOCK_OIDC_CLIENT_ID ?? 'console-app-e2e';
const oidcScope = process.env.MOCK_OIDC_SCOPE ?? 'openid profile email';
const envScript = `window.__ENV__ = {\n` +
  `  OIDC_AUTHORITY: ${JSON.stringify(oidcAuthority)},\n` +
  `  OIDC_CLIENT_ID: ${JSON.stringify(oidcClientId)},\n` +
  `  OIDC_SCOPE: ${JSON.stringify(oidcScope)}\n` +
  `};\n`;
const defaultUserId = 'user-1';
const defaultEmail = 'e2e-tester@agyn.test';

const users = new Map();
const organizations = new Map();
const memberships = new Map();
const secretProviders = new Map();
const secrets = new Map();
const imagePullSecrets = new Map();
const runners = new Map();
const devices = new Map();
const agents = new Map();
const mcps = new Map();
const hooks = new Map();
const llmProviders = new Map();
const models = new Map();
const imagePullSecretAttachments = new Map();
const usageTotals = new Map();
const threads = new Map();
const threadMessages = new Map();
const installations = new Map();
const installationAuditLogs = new Map();

let threadSequence = 0;
let messageSequence = 0;

const defaultUser = {
  id: defaultUserId,
  oidcSubject: 'e2e-oidc-user',
  name: 'E2E Tester',
  email: defaultEmail,
  nickname: 'tester',
  photoUrl: '',
  clusterRole: 'CLUSTER_ROLE_ADMIN',
};

users.set(defaultUserId, defaultUser);

const defaultRunner = {
  id: 'runner-1',
  name: 'Cluster Runner',
  labels: { region: 'local' },
  status: 'RUNNER_STATUS_ENROLLED',
  identityId: 'runner-identity',
  organizationId: '',
  openzitiServiceName: 'cluster-runner',
};

runners.set(defaultRunner.id, defaultRunner);

const defaultLlmProvider = {
  id: 'llm-provider-e2e',
  endpoint: 'https://llm.e2e.agyn.dev',
  authMethod: 'AUTH_METHOD_BEARER',
  organizationId: '',
  token: 'e2e-token',
  createdAt: new Date().toISOString(),
};

llmProviders.set(defaultLlmProvider.id, defaultLlmProvider);

const defaultModel = {
  id: 'model-e2e',
  name: 'E2E Model',
  llmProviderId: 'llm-provider-e2e',
  remoteName: 'gpt-4o-mini',
  organizationId: defaultLlmProvider.organizationId,
  createdAt: new Date().toISOString(),
};

models.set(defaultModel.id, defaultModel);

const defaultInstallation = {
  id: 'installation-1',
  appId: 'app-1',
  organizationId: '',
  slug: 'default-installation',
  configuration: {},
  status: 'Installation ready.',
  createdAt: new Date().toISOString(),
};

installations.set(defaultInstallation.id, defaultInstallation);

const defaultInstallationAuditLogs = [
  {
    id: 'installation-log-1',
    installationId: defaultInstallation.id,
    message: 'Installation created.',
    level: 'INSTALLATION_AUDIT_LOG_LEVEL_INFO',
    createdAt: new Date().toISOString(),
  },
];

installationAuditLogs.set(defaultInstallation.id, defaultInstallationAuditLogs);

function setCors(res) {
  res.setHeader('Access-Control-Allow-Origin', '*');
  res.setHeader('Access-Control-Allow-Headers', 'Authorization, Content-Type, Connect-Protocol-Version');
  res.setHeader('Access-Control-Allow-Methods', 'POST, GET, OPTIONS');
}

function sendJson(res, status, body) {
  setCors(res);
  res.statusCode = status;
  res.setHeader('Content-Type', 'application/json');
  res.end(JSON.stringify(body));
}

function sendText(res, status, body) {
  setCors(res);
  res.statusCode = status;
  res.setHeader('Content-Type', 'text/plain');
  res.end(body);
}

function base64UrlEncode(input) {
  return Buffer.from(JSON.stringify(input))
    .toString('base64')
    .replace(/\+/g, '-')
    .replace(/\//g, '_')
    .replace(/=+$/g, '');
}

function createJwt(payload) {
  const header = base64UrlEncode({ alg: 'none', typ: 'JWT' });
  const encodedPayload = base64UrlEncode(payload);
  return `${header}.${encodedPayload}.`;
}

function proxyRequest(req, res) {
  const targetUrl = new URL(req.url ?? '/', proxyTarget);
  const proxy = http.request(
    {
      hostname: targetUrl.hostname,
      port: targetUrl.port,
      path: `${targetUrl.pathname}${targetUrl.search}`,
      method: req.method,
      headers: req.headers,
    },
    (proxyRes) => {
      res.writeHead(proxyRes.statusCode ?? 502, proxyRes.headers);
      proxyRes.pipe(res);
    },
  );
  proxy.on('error', () => {
    sendText(res, 502, 'Bad gateway');
  });
  req.pipe(proxy);
}

async function parseBody(req) {
  const chunks = [];
  for await (const chunk of req) {
    chunks.push(chunk);
  }
  const raw = Buffer.concat(chunks).toString('utf8');
  if (!raw) return {};
  const contentType = req.headers['content-type'] ?? '';
  if (contentType.includes('application/json')) {
    try {
      return JSON.parse(raw);
    } catch {
      return {};
    }
  }
  if (contentType.includes('application/x-www-form-urlencoded')) {
    return Object.fromEntries(new URLSearchParams(raw));
  }
  return {};
}

function normalizeMembershipRole(value) {
  if (typeof value === 'string') return value;
  if (value === 1) return 'MEMBERSHIP_ROLE_OWNER';
  if (value === 2) return 'MEMBERSHIP_ROLE_MEMBER';
  return 'MEMBERSHIP_ROLE_UNSPECIFIED';
}

function normalizeMembershipStatus(value) {
  if (typeof value === 'string') return value;
  if (value === 1) return 'MEMBERSHIP_STATUS_PENDING';
  if (value === 2) return 'MEMBERSHIP_STATUS_ACTIVE';
  return 'MEMBERSHIP_STATUS_UNSPECIFIED';
}

function normalizeClusterRole(value) {
  if (typeof value === 'string') return value;
  if (value === 1) return 'CLUSTER_ROLE_ADMIN';
  return 'CLUSTER_ROLE_UNSPECIFIED';
}

function normalizeThreadStatus(value) {
  if (typeof value === 'string') return value;
  if (value === 1) return 'THREAD_STATUS_ACTIVE';
  if (value === 2) return 'THREAD_STATUS_ARCHIVED';
  return 'THREAD_STATUS_UNSPECIFIED';
}

function normalizeMessageOrder(value) {
  if (typeof value === 'string') return value;
  if (value === 1) return 'MESSAGE_ORDER_OLDEST_FIRST';
  if (value === 2) return 'MESSAGE_ORDER_NEWEST_FIRST';
  return 'MESSAGE_ORDER_UNSPECIFIED';
}

function normalizeGranularity(value) {
  if (typeof value === 'string') return value;
  if (value === 1) return 'GRANULARITY_TOTAL';
  if (value === 2) return 'GRANULARITY_DAY';
  return 'GRANULARITY_UNSPECIFIED';
}

function resolveGroupValue(groupBy) {
  if (!groupBy) return '';
  if (groupBy === 'status') return 'success';
  if (groupBy === 'kind') return 'input';
  if (groupBy === 'identity_id') return defaultUserId;
  if (groupBy === 'resource_id') return defaultModel.id;
  return 'unknown';
}

function normalizeInt64(value) {
  if (typeof value === 'bigint') return value;
  if (typeof value === 'number') return BigInt(value);
  if (typeof value === 'string') return BigInt(value);
  return 0n;
}

function parsePageToken(token) {
  if (!token) return 0;
  const parsed = Number.parseInt(token, 10);
  return Number.isNaN(parsed) ? 0 : parsed;
}

function paginate(items, pageSize, pageToken) {
  const size = Number(pageSize) > 0 ? Number(pageSize) : items.length;
  const start = parsePageToken(pageToken);
  const end = start + size;
  return {
    items: items.slice(start, end),
    nextPageToken: end < items.length ? String(end) : '',
  };
}

function recordUsageTotal(orgId, value) {
  if (!orgId) return;
  const total = usageTotals.get(orgId) ?? 0n;
  usageTotals.set(orgId, total + value);
}

function parseUsageRecord(recordBytes) {
  const reader = new BinaryReader(recordBytes);
  let orgId = '';
  let value = 0n;

  while (reader.pos < reader.len) {
    const [fieldNo, wireType] = reader.tag();
    if (fieldNo === 1 && wireType === WireType.LengthDelimited) {
      orgId = reader.string();
      continue;
    }
    if (fieldNo === 7 && wireType === WireType.Varint) {
      value = normalizeInt64(reader.int64());
      continue;
    }
    reader.skip(wireType, fieldNo);
  }

  recordUsageTotal(orgId, value);
}

function parseRecordRequest(messageBytes) {
  const reader = new BinaryReader(messageBytes);
  while (reader.pos < reader.len) {
    const [fieldNo, wireType] = reader.tag();
    if (fieldNo === 1 && wireType === WireType.LengthDelimited) {
      parseUsageRecord(reader.bytes());
      continue;
    }
    reader.skip(wireType, fieldNo);
  }
}

function parseGrpcRecordPayload(buffer) {
  let offset = 0;
  while (offset + 5 <= buffer.length) {
    const compressed = buffer[offset];
    const messageLength = buffer.readUInt32BE(offset + 1);
    const start = offset + 5;
    const end = start + messageLength;
    if (end > buffer.length) return;
    if (compressed === 0) {
      parseRecordRequest(buffer.subarray(start, end));
    }
    offset = end;
  }
}

function mapUser(user) {
  return {
    meta: { id: user.id },
    oidcSubject: user.oidcSubject,
    name: user.name,
    email: user.email,
    nickname: user.nickname,
    photoUrl: user.photoUrl,
  };
}

function mapMembership(membership) {
  return {
    id: membership.id,
    organizationId: membership.organizationId,
    identityId: membership.identityId,
    role: membership.role,
    status: membership.status,
  };
}

function mapRunner(runner) {
  return {
    meta: { id: runner.id },
    name: runner.name,
    labels: runner.labels,
    status: runner.status,
    identityId: runner.identityId,
    organizationId: runner.organizationId,
    openzitiServiceName: runner.openzitiServiceName,
  };
}

function mapSecretProvider(provider) {
  return {
    meta: { id: provider.id },
    title: provider.title,
    description: provider.description,
    type: provider.type,
    config: provider.config,
    organizationId: provider.organizationId,
  };
}

function mapSecret(secret) {
  return {
    meta: { id: secret.id },
    title: secret.title,
    description: secret.description,
    secretProviderId: secret.secretProviderId,
    remoteName: secret.remoteName,
    organizationId: secret.organizationId,
  };
}

function mapEntityMeta(entity) {
  return {
    id: entity.id,
    createdAt: entity.createdAt,
  };
}

function mapDevice(device) {
  return {
    meta: mapEntityMeta(device),
    userIdentityId: device.userIdentityId,
    name: device.name,
    openzitiIdentityId: device.openzitiIdentityId,
    status: device.status,
  };
}

function mapAgent(agent) {
  return {
    meta: mapEntityMeta(agent),
    name: agent.name,
    nickname: agent.nickname,
    role: agent.role,
    model: agent.model,
    description: agent.description,
    configuration: agent.configuration,
    image: agent.image,
    initImage: agent.initImage,
    resources: agent.resources,
    organizationId: agent.organizationId,
  };
}

function mapMcp(mcp) {
  return {
    meta: mapEntityMeta(mcp),
    agentId: mcp.agentId,
    image: mcp.image,
    command: mcp.command,
    description: mcp.description,
    name: mcp.name,
    resources: mcp.resources,
  };
}

function mapHook(hook) {
  return {
    meta: mapEntityMeta(hook),
    agentId: hook.agentId,
    event: hook.event,
    function: hook.functionName,
    image: hook.image,
    description: hook.description,
    resources: hook.resources,
  };
}

function mapModel(model) {
  return {
    meta: mapEntityMeta(model),
    name: model.name,
    llmProviderId: model.llmProviderId,
    remoteName: model.remoteName,
  };
}

function mapLlmProvider(provider) {
  return {
    meta: mapEntityMeta(provider),
    endpoint: provider.endpoint,
    authMethod: provider.authMethod,
    organizationId: provider.organizationId,
  };
}

function mapImagePullSecret(secret) {
  return {
    meta: mapEntityMeta(secret),
    description: secret.description,
    registry: secret.registry,
    username: secret.username,
    organizationId: secret.organizationId,
  };
}

function mapImagePullSecretAttachment(attachment) {
  return {
    meta: mapEntityMeta(attachment),
    imagePullSecretId: attachment.imagePullSecretId,
    target: attachment.target,
  };
}

function mapThreadParticipant(participant) {
  return {
    id: participant.id,
    joinedAt: participant.joinedAt,
    passive: Boolean(participant.passive),
  };
}

function mapThread(thread) {
  return {
    id: thread.id,
    participants: thread.participants.map(mapThreadParticipant),
    status: thread.status,
    createdAt: thread.createdAt,
    updatedAt: thread.updatedAt,
    organizationId: thread.organizationId,
    messageCount: thread.messageCount,
  };
}

function mapThreadMessage(message) {
  return {
    id: message.id,
    threadId: message.threadId,
    senderId: message.senderId,
    body: message.body,
    fileIds: message.fileIds ?? [],
    createdAt: message.createdAt,
  };
}

function mapInstallation(installation) {
  return {
    meta: mapEntityMeta(installation),
    appId: installation.appId,
    organizationId: installation.organizationId,
    slug: installation.slug,
    configuration: installation.configuration,
    status: installation.status ?? '',
  };
}

function mapInstallationAuditLogEntry(entry) {
  return {
    id: entry.id,
    installationId: entry.installationId,
    message: entry.message,
    level: entry.level,
    createdAt: entry.createdAt,
  };
}

function handleUsersGateway(method, body, res) {
  switch (method) {
    case 'GetMe': {
      return sendJson(res, 200, { user: mapUser(defaultUser), clusterRole: defaultUser.clusterRole });
    }
    case 'ListUsers': {
      return sendJson(res, 200, {
        users: Array.from(users.values()).map(mapUser),
        nextPageToken: '',
      });
    }
    case 'BatchGetUsers': {
      const ids = Array.isArray(body.identityIds) ? body.identityIds : [];
      const result = ids
        .map((id) => users.get(id))
        .filter(Boolean)
        .map(mapUser);
      return sendJson(res, 200, { users: result });
    }
    case 'CreateUser': {
      const id = randomUUID();
      const user = {
        id,
        oidcSubject: body.oidcSubject ?? `mock-${id}`,
        name: body.name ?? body.oidcSubject ?? id,
        email: body.email ?? '',
        nickname: body.nickname ?? '',
        photoUrl: body.photoUrl ?? '',
        clusterRole: normalizeClusterRole(body.clusterRole),
      };
      users.set(id, user);
      return sendJson(res, 200, { user: mapUser(user), clusterRole: user.clusterRole });
    }
    case 'UpdateUser': {
      const identityId = body.identityId;
      if (!identityId || !users.has(identityId)) {
        return sendText(res, 404, 'User not found');
      }
      const user = users.get(identityId);
      if (!user) return sendText(res, 404, 'User not found');
      user.email = body.email ?? user.email;
      user.name = body.name ?? user.name;
      user.nickname = body.nickname ?? user.nickname;
      user.photoUrl = body.photoUrl ?? user.photoUrl;
      if (body.clusterRole !== undefined) {
        user.clusterRole = normalizeClusterRole(body.clusterRole);
      }
      return sendJson(res, 200, { user: mapUser(user), clusterRole: user.clusterRole });
    }
    case 'GetUser': {
      const identityId = body.identityId;
      if (!identityId || !users.has(identityId)) {
        return sendText(res, 404, 'User not found');
      }
      const user = users.get(identityId);
      return sendJson(res, 200, { user: mapUser(user), clusterRole: user.clusterRole });
    }
    case 'DeleteUser': {
      if (body.identityId) users.delete(body.identityId);
      return sendJson(res, 200, {});
    }
    case 'CreateDevice': {
      const id = randomUUID();
      const device = {
        id,
        userIdentityId: defaultUserId,
        name: body.name ?? `device-${id}`,
        openzitiIdentityId: `openziti-${id}`,
        status: 'DEVICE_STATUS_PENDING',
        createdAt: new Date().toISOString(),
      };
      devices.set(id, device);
      return sendJson(res, 200, { device: mapDevice(device), enrollmentJwt: createJwt({ deviceId: id }) });
    }
    case 'ListDevices': {
      return sendJson(res, 200, {
        devices: Array.from(devices.values()).map(mapDevice),
        nextPageToken: '',
      });
    }
    case 'DeleteDevice': {
      if (body.id) devices.delete(body.id);
      return sendJson(res, 200, {});
    }
    default:
      return sendText(res, 404, 'Unknown UsersGateway method');
  }
}

function handleOrganizationsGateway(method, body, res) {
  switch (method) {
    case 'CreateOrganization': {
      const id = randomUUID();
      const name = body.name ?? `org-${id}`;
      const org = { id, name };
      organizations.set(id, org);
      const membership = {
        id: randomUUID(),
        organizationId: id,
        identityId: defaultUserId,
        role: 'MEMBERSHIP_ROLE_OWNER',
        status: 'MEMBERSHIP_STATUS_ACTIVE',
      };
      memberships.set(membership.id, membership);
      return sendJson(res, 200, { organization: org });
    }
    case 'ListOrganizations': {
      return sendJson(res, 200, {
        organizations: Array.from(organizations.values()),
        nextPageToken: '',
      });
    }
    case 'ListAccessibleOrganizations': {
      return sendJson(res, 200, {
        organizations: Array.from(organizations.values()),
        nextPageToken: '',
      });
    }
    case 'ListMyMemberships': {
      const status = normalizeMembershipStatus(body.status);
      const result = Array.from(memberships.values()).filter((membership) => {
        if (membership.identityId !== defaultUserId) return false;
        if (status === 'MEMBERSHIP_STATUS_UNSPECIFIED') return true;
        return membership.status === status;
      });
      return sendJson(res, 200, { memberships: result.map(mapMembership), nextPageToken: '' });
    }
    case 'ListMembers': {
      const status = normalizeMembershipStatus(body.status);
      const result = Array.from(memberships.values()).filter((membership) => {
        if (membership.organizationId !== body.organizationId) return false;
        if (status === 'MEMBERSHIP_STATUS_UNSPECIFIED') return true;
        return membership.status === status;
      });
      return sendJson(res, 200, { memberships: result.map(mapMembership), nextPageToken: '' });
    }
    case 'CreateMembership': {
      const membership = {
        id: randomUUID(),
        organizationId: body.organizationId,
        identityId: body.identityId,
        role: normalizeMembershipRole(body.role),
        status: 'MEMBERSHIP_STATUS_PENDING',
      };
      memberships.set(membership.id, membership);
      return sendJson(res, 200, { membership: mapMembership(membership) });
    }
    case 'UpdateMembershipRole': {
      const membership = memberships.get(body.membershipId);
      if (!membership) return sendText(res, 404, 'Membership not found');
      membership.role = normalizeMembershipRole(body.role);
      return sendJson(res, 200, { membership: mapMembership(membership) });
    }
    case 'RemoveMembership': {
      memberships.delete(body.membershipId);
      return sendJson(res, 200, {});
    }
    default:
      return sendText(res, 404, 'Unknown OrganizationsGateway method');
  }
}

function handleSecretsGateway(method, body, res) {
  switch (method) {
    case 'CreateSecretProvider': {
      const provider = {
        id: randomUUID(),
        title: body.title ?? 'Provider',
        description: body.description ?? '',
        type: body.type ?? 'SECRET_PROVIDER_TYPE_VAULT',
        config: body.config ?? {},
        organizationId: body.organizationId ?? '',
      };
      secretProviders.set(provider.id, provider);
      return sendJson(res, 200, { secretProvider: mapSecretProvider(provider) });
    }
    case 'ListSecretProviders': {
      const providers = Array.from(secretProviders.values()).filter(
        (provider) => provider.organizationId === body.organizationId,
      );
      return sendJson(res, 200, { secretProviders: providers.map(mapSecretProvider), nextPageToken: '' });
    }
    case 'DeleteSecretProvider': {
      if (body.id) {
        secretProviders.delete(body.id);
        for (const [secretId, secret] of secrets.entries()) {
          if (secret.secretProviderId === body.id) {
            secrets.delete(secretId);
          }
        }
      }
      return sendJson(res, 200, {});
    }
    case 'CreateSecret': {
      const secret = {
        id: randomUUID(),
        title: body.title ?? 'Secret',
        description: body.description ?? '',
        secretProviderId: body.secretProviderId ?? '',
        remoteName: body.remoteName ?? '',
        organizationId: body.organizationId ?? '',
      };
      secrets.set(secret.id, secret);
      return sendJson(res, 200, { secret: mapSecret(secret) });
    }
    case 'ListSecrets': {
      const providerId = body.secretProviderId || '';
      const result = Array.from(secrets.values()).filter((secret) => {
        if (secret.organizationId !== body.organizationId) return false;
        if (!providerId) return true;
        return secret.secretProviderId === providerId;
      });
      return sendJson(res, 200, { secrets: result.map(mapSecret), nextPageToken: '' });
    }
    case 'DeleteSecret': {
      if (body.id) secrets.delete(body.id);
      return sendJson(res, 200, {});
    }
    case 'CreateImagePullSecret': {
      const id = randomUUID();
      const secret = {
        id,
        description: body.description ?? '',
        registry: body.registry ?? '',
        username: body.username ?? '',
        organizationId: body.organizationId ?? '',
        createdAt: new Date().toISOString(),
      };
      imagePullSecrets.set(id, secret);
      return sendJson(res, 200, { imagePullSecret: mapImagePullSecret(secret) });
    }
    case 'ListImagePullSecrets': {
      const result = Array.from(imagePullSecrets.values()).filter(
        (secret) => secret.organizationId === body.organizationId,
      );
      return sendJson(res, 200, { imagePullSecrets: result.map(mapImagePullSecret), nextPageToken: '' });
    }
    case 'UpdateImagePullSecret': {
      const secret = imagePullSecrets.get(body.id);
      if (!secret) return sendText(res, 404, 'Image pull secret not found');
      secret.description = body.description ?? secret.description;
      secret.registry = body.registry ?? secret.registry;
      secret.username = body.username ?? secret.username;
      return sendJson(res, 200, { imagePullSecret: mapImagePullSecret(secret) });
    }
    case 'DeleteImagePullSecret': {
      if (body.id) {
        imagePullSecrets.delete(body.id);
        for (const [attachmentId, attachment] of imagePullSecretAttachments.entries()) {
          if (attachment.imagePullSecretId === body.id) {
            imagePullSecretAttachments.delete(attachmentId);
          }
        }
      }
      return sendJson(res, 200, {});
    }
    default:
      return sendText(res, 404, 'Unknown SecretsGateway method');
  }
}

function handleRunnersGateway(method, body, res) {
  switch (method) {
    case 'RegisterRunner': {
      const id = randomUUID();
      const runner = {
        id,
        name: body.name ?? `runner-${id}`,
        labels: body.labels ?? {},
        status: 'RUNNER_STATUS_ENROLLED',
        identityId: `runner-identity-${id}`,
        organizationId: body.organizationId ?? '',
        openzitiServiceName: `runner-${id}`,
      };
      runners.set(id, runner);
      return sendJson(res, 200, { runner: mapRunner(runner), serviceToken: `token-${id}` });
    }
    case 'ListRunners': {
      const orgId = body.organizationId ?? '';
      const result = Array.from(runners.values()).filter((runner) => {
        if (!orgId) return !runner.organizationId;
        return runner.organizationId === orgId || !runner.organizationId;
      });
      return sendJson(res, 200, { runners: result.map(mapRunner), nextPageToken: '' });
    }
    case 'GetRunner': {
      const runner = runners.get(body.id);
      if (!runner) return sendText(res, 404, 'Runner not found');
      return sendJson(res, 200, { runner: mapRunner(runner) });
    }
    case 'ListWorkloads': {
      return sendJson(res, 200, { workloads: [], nextPageToken: '' });
    }
    case 'UpdateRunner': {
      const runner = runners.get(body.id);
      if (!runner) return sendText(res, 404, 'Runner not found');
      runner.labels = body.labels ?? runner.labels;
      return sendJson(res, 200, { runner: mapRunner(runner) });
    }
    case 'DeleteRunner': {
      if (body.id) runners.delete(body.id);
      return sendJson(res, 200, {});
    }
    default:
      return sendText(res, 404, 'Unknown RunnersGateway method');
  }
}

function handleAgentsGateway(method, body, res) {
  switch (method) {
    case 'CreateAgent': {
      const id = randomUUID();
      const agent = {
        id,
        name: body.name ?? `agent-${id}`,
        nickname: body.nickname ?? '',
        role: body.role ?? '',
        model: body.model ?? '',
        description: body.description ?? '',
        configuration: body.configuration ?? '',
        image: body.image ?? '',
        initImage: body.initImage ?? '',
        resources: body.resources,
        organizationId: body.organizationId ?? '',
        createdAt: new Date().toISOString(),
      };
      agents.set(id, agent);
      return sendJson(res, 200, { agent: mapAgent(agent) });
    }
    case 'GetAgent': {
      const agent = agents.get(body.id);
      if (!agent) return sendText(res, 404, 'Agent not found');
      return sendJson(res, 200, { agent: mapAgent(agent) });
    }
    case 'ListAgents': {
      const orgId = body.organizationId ?? '';
      const result = Array.from(agents.values()).filter((agent) => !orgId || agent.organizationId === orgId);
      return sendJson(res, 200, { agents: result.map(mapAgent), nextPageToken: '' });
    }
    case 'DeleteAgent': {
      if (body.id) {
        agents.delete(body.id);
        for (const [mcpId, mcp] of mcps.entries()) {
          if (mcp.agentId === body.id) mcps.delete(mcpId);
        }
        for (const [hookId, hook] of hooks.entries()) {
          if (hook.agentId === body.id) hooks.delete(hookId);
        }
        for (const [attachmentId, attachment] of imagePullSecretAttachments.entries()) {
          if (attachment.target.case === 'agentId' && attachment.target.value === body.id) {
            imagePullSecretAttachments.delete(attachmentId);
          }
        }
      }
      return sendJson(res, 200, {});
    }
    case 'CreateMcp': {
      const id = randomUUID();
      const mcp = {
        id,
        agentId: body.agentId ?? '',
        image: body.image ?? '',
        command: body.command ?? '',
        description: body.description ?? '',
        name: body.name ?? `mcp-${id}`,
        resources: body.resources,
        createdAt: new Date().toISOString(),
      };
      mcps.set(id, mcp);
      return sendJson(res, 200, { mcp: mapMcp(mcp) });
    }
    case 'ListMcps': {
      const agentId = body.agentId ?? '';
      const result = Array.from(mcps.values()).filter((mcp) => !agentId || mcp.agentId === agentId);
      return sendJson(res, 200, { mcps: result.map(mapMcp), nextPageToken: '' });
    }
    case 'CreateHook': {
      const id = randomUUID();
      const hook = {
        id,
        agentId: body.agentId ?? '',
        event: body.event ?? '',
        functionName: body.function ?? '',
        image: body.image ?? '',
        description: body.description ?? '',
        resources: body.resources,
        createdAt: new Date().toISOString(),
      };
      hooks.set(id, hook);
      return sendJson(res, 200, { hook: mapHook(hook) });
    }
    case 'ListHooks': {
      const agentId = body.agentId ?? '';
      const result = Array.from(hooks.values()).filter((hook) => !agentId || hook.agentId === agentId);
      return sendJson(res, 200, { hooks: result.map(mapHook), nextPageToken: '' });
    }
    case 'CreateImagePullSecretAttachment': {
      const imagePullSecretId = body.imagePullSecretId ?? '';
      let target = body.target;
      if (!target?.case || !target?.value) {
        if (body.agentId) {
          target = { case: 'agentId', value: body.agentId };
        } else if (body.mcpId) {
          target = { case: 'mcpId', value: body.mcpId };
        } else if (body.hookId) {
          target = { case: 'hookId', value: body.hookId };
        }
      }
      if (!imagePullSecretId || !target?.case || !target?.value) {
        return sendText(res, 400, 'Missing image pull secret attachment target');
      }
      const alreadyExists = Array.from(imagePullSecretAttachments.values()).some(
        (attachment) =>
          attachment.imagePullSecretId === imagePullSecretId &&
          attachment.target.case === target.case &&
          attachment.target.value === target.value,
      );
      if (alreadyExists) {
        return sendText(res, 409, 'Image pull secret attachment already exists');
      }
      const id = randomUUID();
      const attachment = {
        id,
        imagePullSecretId,
        target: { case: target.case, value: target.value },
        createdAt: new Date().toISOString(),
      };
      imagePullSecretAttachments.set(id, attachment);
      return sendJson(res, 200, { imagePullSecretAttachment: mapImagePullSecretAttachment(attachment) });
    }
    case 'ListImagePullSecretAttachments': {
      const result = Array.from(imagePullSecretAttachments.values()).filter((attachment) => {
        if (body.imagePullSecretId && attachment.imagePullSecretId !== body.imagePullSecretId) {
          return false;
        }
        if (body.agentId) {
          return attachment.target.case === 'agentId' && attachment.target.value === body.agentId;
        }
        if (body.mcpId) {
          return attachment.target.case === 'mcpId' && attachment.target.value === body.mcpId;
        }
        if (body.hookId) {
          return attachment.target.case === 'hookId' && attachment.target.value === body.hookId;
        }
        return true;
      });
      return sendJson(res, 200, { imagePullSecretAttachments: result.map(mapImagePullSecretAttachment), nextPageToken: '' });
    }
    case 'DeleteImagePullSecretAttachment': {
      if (body.id) imagePullSecretAttachments.delete(body.id);
      return sendJson(res, 200, {});
    }
    case 'ListSkills': {
      return sendJson(res, 200, { skills: [], nextPageToken: '' });
    }
    case 'ListEnvs': {
      return sendJson(res, 200, { envs: [], nextPageToken: '' });
    }
    case 'ListInitScripts': {
      return sendJson(res, 200, { initScripts: [], nextPageToken: '' });
    }
    case 'ListVolumeAttachments': {
      return sendJson(res, 200, { volumeAttachments: [], nextPageToken: '' });
    }
    case 'ListVolumes': {
      return sendJson(res, 200, { volumes: [], nextPageToken: '' });
    }
    default:
      return sendText(res, 404, 'Unknown AgentsGateway method');
  }
}

function handleThreadsGateway(method, body, res) {
  switch (method) {
    case 'CreateThread': {
      const id = randomUUID();
      const createdAt = new Date(Date.now() + threadSequence).toISOString();
      threadSequence += 1000;
      const participantIds = Array.isArray(body.participantIds) ? body.participantIds : [];
      const participantIdentifiers = Array.isArray(body.participants) ? body.participants : [];
      const resolvedIds = participantIdentifiers
        .map((participant) => participant.participantId || participant.participantNickname)
        .filter(Boolean);
      const uniqueIds = Array.from(new Set([...participantIds, ...resolvedIds].filter(Boolean)));
      const participants = uniqueIds.map((participantId) => ({
        id: participantId,
        joinedAt: createdAt,
        passive: false,
      }));
      const thread = {
        id,
        participants,
        status: 'THREAD_STATUS_ACTIVE',
        createdAt,
        updatedAt: createdAt,
        organizationId: body.organizationId ?? '',
        messageCount: 0,
      };
      threads.set(id, thread);
      threadMessages.set(id, []);
      return sendJson(res, 200, { thread: mapThread(thread) });
    }
    case 'ArchiveThread': {
      const thread = threads.get(body.threadId);
      if (!thread) return sendText(res, 404, 'Thread not found');
      thread.status = 'THREAD_STATUS_ARCHIVED';
      thread.updatedAt = new Date(Date.now() + threadSequence).toISOString();
      threadSequence += 1000;
      return sendJson(res, 200, { thread: mapThread(thread) });
    }
    case 'AddParticipant': {
      const thread = threads.get(body.threadId);
      if (!thread) return sendText(res, 404, 'Thread not found');
      const participantId =
        body.participantId ?? body.participant?.participantId ?? body.participant?.participantNickname ?? '';
      if (!participantId) return sendText(res, 400, 'Missing participant id');
      if (!thread.participants.some((participant) => participant.id === participantId)) {
        thread.participants.push({
          id: participantId,
          joinedAt: new Date(Date.now() + threadSequence).toISOString(),
          passive: Boolean(body.passive),
        });
        threadSequence += 1000;
      }
      return sendJson(res, 200, { thread: mapThread(thread) });
    }
    case 'SendMessage': {
      const threadId = body.threadId ?? '';
      const thread = threads.get(threadId);
      if (!thread) return sendText(res, 404, 'Thread not found');
      const createdAt = new Date(Date.now() + messageSequence).toISOString();
      messageSequence += 1;
      const message = {
        id: randomUUID(),
        threadId,
        senderId: body.senderId ?? '',
        body: body.body ?? '',
        fileIds: Array.isArray(body.fileIds) ? body.fileIds : [],
        createdAt,
      };
      const messages = threadMessages.get(threadId) ?? [];
      messages.push(message);
      threadMessages.set(threadId, messages);
      thread.messageCount = messages.length;
      thread.updatedAt = createdAt;
      if (message.senderId && !thread.participants.some((participant) => participant.id === message.senderId)) {
        thread.participants.push({
          id: message.senderId,
          joinedAt: createdAt,
          passive: false,
        });
      }
      return sendJson(res, 200, { message: mapThreadMessage(message) });
    }
    case 'GetThreads': {
      const participantId = body.participantId ?? '';
      let result = Array.from(threads.values());
      if (participantId) {
        result = result.filter((thread) =>
          thread.participants.some((participant) => participant.id === participantId),
        );
      }
      result.sort((a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime());
      const { items, nextPageToken } = paginate(result, body.pageSize, body.pageToken);
      return sendJson(res, 200, { threads: items.map(mapThread), nextPageToken });
    }
    case 'GetOrganizationThreads': {
      const orgId = body.organizationId ?? '';
      const statusFilter = normalizeThreadStatus(body.status);
      let result = Array.from(threads.values()).filter((thread) => thread.organizationId === orgId);
      if (statusFilter !== 'THREAD_STATUS_UNSPECIFIED') {
        result = result.filter((thread) => thread.status === statusFilter);
      }
      result.sort((a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime());
      const { items, nextPageToken } = paginate(result, body.pageSize, body.pageToken);
      return sendJson(res, 200, { threads: items.map(mapThread), nextPageToken });
    }
    case 'GetThread': {
      const thread = threads.get(body.threadId);
      if (!thread) return sendText(res, 404, 'Thread not found');
      return sendJson(res, 200, { thread: mapThread(thread) });
    }
    case 'GetMessages': {
      const threadId = body.threadId ?? '';
      const messages = threadMessages.get(threadId);
      if (!messages) return sendText(res, 404, 'Thread not found');
      const order = normalizeMessageOrder(body.order);
      const ordered = [...messages].sort((a, b) => {
        const aTime = new Date(a.createdAt).getTime();
        const bTime = new Date(b.createdAt).getTime();
        if (order === 'MESSAGE_ORDER_NEWEST_FIRST') return bTime - aTime;
        return aTime - bTime;
      });
      const { items, nextPageToken } = paginate(ordered, body.pageSize, body.pageToken);
      return sendJson(res, 200, { messages: items.map(mapThreadMessage), nextPageToken });
    }
    default:
      return sendText(res, 404, 'Unknown ThreadsGateway method');
  }
}

function handleAppsGateway(method, body, res) {
  if (method === 'ListInstallations') {
    return sendJson(res, 200, {
      installations: Array.from(installations.values()).map(mapInstallation),
      nextPageToken: '',
    });
  }
  if (method === 'ListInstallationAuditLogEntries') {
    const installationId = body.installationId ?? body.installation_id ?? '';
    const entries = installationAuditLogs.get(installationId) ?? [];
    const { items, nextPageToken } = paginate(entries, body.pageSize, body.pageToken);
    return sendJson(res, 200, {
      entries: items.map(mapInstallationAuditLogEntry),
      nextPageToken,
    });
  }
  if (method === 'ReportInstallationStatus') {
    const installationId = body.installationId ?? body.installation_id ?? '';
    const status = body.status ?? '';
    const existing = installations.get(installationId);
    if (installationId) {
      if (existing) {
        existing.status = status;
      } else {
        installations.set(installationId, {
          ...defaultInstallation,
          id: installationId,
          status,
        });
      }
    }
    const installation = installations.get(installationId) ?? defaultInstallation;
    return sendJson(res, 200, {
      installation: mapInstallation(installation),
    });
  }
  if (method === 'AppendInstallationAuditLogEntry') {
    const installationId = body.installationId ?? body.installation_id ?? defaultInstallation.id;
    const entry = {
      id: randomUUID(),
      installationId,
      message: body.message ?? '',
      level: body.level ?? 'INSTALLATION_AUDIT_LOG_LEVEL_INFO',
      createdAt: new Date().toISOString(),
    };
    const entries = installationAuditLogs.get(installationId) ?? [];
    entries.unshift(entry);
    installationAuditLogs.set(installationId, entries);
    return sendJson(res, 200, {
      entry: mapInstallationAuditLogEntry(entry),
    });
  }
  return sendText(res, 404, 'Unknown AppsGateway method');
}

function handleMeteringGateway(method, body, res) {
  if (method !== 'QueryUsage') {
    return sendText(res, 404, 'Unknown MeteringGateway method');
  }
  const orgId = body.orgId ?? body.org_id ?? '';
  const total = usageTotals.get(orgId) ?? 0n;
  if (!orgId || total === 0n) {
    return sendJson(res, 200, { buckets: [] });
  }
  const granularity = normalizeGranularity(body.granularity);
  const groupValue = resolveGroupValue(body.groupBy ?? '');
  const bucket = {
    value: total.toString(),
    groupValue,
  };
  if (granularity === 'GRANULARITY_DAY') {
    bucket.timestamp = new Date().toISOString();
  }
  return sendJson(res, 200, { buckets: [bucket] });
}

async function handleLlmGateway(method, body, res) {
  switch (method) {
    case 'ListLLMProviders': {
      const organizationId = body.organizationId ?? '';
      const result = Array.from(llmProviders.values()).filter(
        (provider) => !organizationId || provider.organizationId === organizationId,
      );
      return sendJson(res, 200, { providers: result.map(mapLlmProvider), nextPageToken: '' });
    }
    case 'CreateLLMProvider': {
      const provider = {
        id: randomUUID(),
        endpoint: body.endpoint ?? '',
        authMethod: body.authMethod ?? 'AUTH_METHOD_BEARER',
        organizationId: body.organizationId ?? '',
        token: body.token ?? '',
        createdAt: new Date().toISOString(),
      };
      llmProviders.set(provider.id, provider);
      return sendJson(res, 200, { provider: mapLlmProvider(provider) });
    }
    case 'CreateModel': {
      const model = {
        id: randomUUID(),
        name: body.name ?? '',
        llmProviderId: body.llmProviderId ?? '',
        remoteName: body.remoteName ?? '',
        organizationId: body.organizationId ?? '',
        createdAt: new Date().toISOString(),
      };
      models.set(model.id, model);
      return sendJson(res, 200, { model: mapModel(model) });
    }
    case 'ListModels': {
      const providerId = body.llmProviderId ?? '';
      const organizationId = body.organizationId ?? '';
      const result = Array.from(models.values()).filter((model) => {
        if (providerId && model.llmProviderId !== providerId) return false;
        if (organizationId && model.organizationId !== organizationId) return false;
        return true;
      });
      return sendJson(res, 200, { models: result.map(mapModel), nextPageToken: '' });
    }
    case 'TestModel': {
      const modelId = body.modelId ?? '';
      const model = modelId ? models.get(modelId) : null;
      if (!model) {
        return sendText(res, 404, 'Model not found');
      }
      if (model.remoteName === 'nonexistent-model') {
        return sendText(res, 400, 'Model test failed');
      }
      const label = model.remoteName || model.name || 'model';
      await new Promise((resolve) => setTimeout(resolve, 200));
      return sendJson(res, 200, { outputText: `Test response from ${label}.` });
    }
    default:
      return sendText(res, 404, 'Unknown LLMGateway method');
  }
}

const grpcEmptyMessage = Buffer.from([0, 0, 0, 0, 0]);

function respondGrpc(stream, status, message) {
  const headers = {
    ':status': 200,
    'content-type': 'application/grpc+proto',
    'grpc-status': String(status),
  };
  if (message) headers['grpc-message'] = message;
  stream.respond(headers);
  stream.end(grpcEmptyMessage);
}

const server = http.createServer(async (req, res) => {
  setCors(res);
  if (req.method === 'OPTIONS') {
    res.statusCode = 204;
    res.end();
    return;
  }

  const url = new URL(req.url ?? '/', `http://${req.headers.host}`);
  const pathname = url.pathname;

  if (pathname === '/healthz') {
    return sendText(res, 200, 'ok');
  }

  if (pathname === '/env.js') {
    res.statusCode = 200;
    res.setHeader('Content-Type', 'application/javascript');
    res.end(envScript);
    return;
  }

  if (pathname === '/authorize') {
    const redirectUri = url.searchParams.get('redirect_uri');
    if (!redirectUri) return sendText(res, 400, 'missing redirect_uri');
    const state = url.searchParams.get('state');
    const code = randomUUID();
    const redirect = new URL(redirectUri);
    redirect.searchParams.set('code', code);
    if (state) redirect.searchParams.set('state', state);
    res.statusCode = 302;
    res.setHeader('Location', redirect.toString());
    res.end();
    return;
  }

  if (pathname === '/token') {
    const token = randomUUID();
    const now = Math.floor(Date.now() / 1000);
    const idToken = createJwt({
      sub: defaultUserId,
      iss: oidcAuthority,
      aud: oidcClientId,
      exp: now + 3600,
      iat: now,
      name: defaultUser.name,
      email: defaultUser.email,
    });
    return sendJson(res, 200, {
      access_token: `access-${token}`,
      id_token: idToken,
      refresh_token: `refresh-${token}`,
      token_type: 'Bearer',
      scope: oidcScope,
      expires_in: 3600,
      session_state: randomUUID(),
    });
  }

  if (pathname === '/end-session') {
    const redirectUri = url.searchParams.get('post_logout_redirect_uri');
    if (!redirectUri) {
      return sendText(res, 400, 'missing post_logout_redirect_uri');
    }
    res.statusCode = 302;
    res.setHeader('Location', redirectUri);
    res.end();
    return;
  }

  if (pathname === '/jwks.json') {
    return sendJson(res, 200, { keys: [] });
  }

  if (pathname === '/userinfo') {
    return sendJson(res, 200, {
      sub: defaultUserId,
      name: defaultUser.name,
      email: defaultUser.email,
    });
  }

  if (pathname === '/api/test/client-auth-strategies') {
    if (req.method === 'POST') {
      return sendJson(res, 200, {});
    }
  }

  if (pathname.startsWith('/api/')) {
    const body = await parseBody(req);
    const parts = pathname.split('/').filter(Boolean);
    const service = parts[1];
    const method = parts[2];
    if (!service || !method) return sendText(res, 404, 'Invalid gateway path');
    if (service === 'agynio.api.gateway.v1.UsersGateway') {
      return handleUsersGateway(method, body, res);
    }
    if (service === 'agynio.api.gateway.v1.OrganizationsGateway') {
      return handleOrganizationsGateway(method, body, res);
    }
    if (service === 'agynio.api.gateway.v1.SecretsGateway') {
      return handleSecretsGateway(method, body, res);
    }
    if (service === 'agynio.api.gateway.v1.RunnersGateway') {
      return handleRunnersGateway(method, body, res);
    }
    if (service === 'agynio.api.gateway.v1.AgentsGateway') {
      return handleAgentsGateway(method, body, res);
    }
    if (service === 'agynio.api.gateway.v1.ThreadsGateway') {
      return handleThreadsGateway(method, body, res);
    }
    if (service === 'agynio.api.gateway.v1.MeteringGateway') {
      return handleMeteringGateway(method, body, res);
    }
    if (service === 'agynio.api.gateway.v1.LLMGateway') {
      await handleLlmGateway(method, body, res);
      return;
    }
    if (service === 'agynio.api.gateway.v1.AppsGateway') {
      return handleAppsGateway(method, body, res);
    }
    return sendText(res, 404, 'Unknown gateway');
  }

  return proxyRequest(req, res);
});

server.listen(port, () => {
  console.log(`[mock-server] listening on ${port}`);
});

const meteringServer = http2.createServer();

meteringServer.on('stream', (stream, headers) => {
  const path = headers[':path'];
  if (path === '/agynio.api.metering.v1.MeteringService/Record') {
    const chunks = [];
    stream.on('data', (chunk) => chunks.push(chunk));
    stream.on('end', () => {
      try {
        const payload = Buffer.concat(chunks);
        if (payload.length > 0) {
          parseGrpcRecordPayload(payload);
        }
      } catch {
        // ignore malformed payloads in mock
      }
      respondGrpc(stream, 0);
    });
    return;
  }

  respondGrpc(stream, 12, 'unimplemented');
});

meteringServer.listen(meteringPort, () => {
  console.log(`[mock-metering] listening on ${meteringPort}`);
});
