import { randomBytes } from 'node:crypto';
import { expect, test, type APIRequestContext, type Browser } from '@playwright/test';
import { signInViaOidc } from './sign-in-helper';
import { readOidcSession } from './oidc-helpers';

const USERS_GATEWAY_PATH = '/api/agynio.api.gateway.v1.UsersGateway';
const ORGS_GATEWAY_PATH = '/api/agynio.api.gateway.v1.OrganizationsGateway';
const THREADS_GATEWAY_PATH = '/api/agynio.api.gateway.v1.ThreadsGateway';

const CONNECT_HEADERS = {
  'Content-Type': 'application/json',
  'Connect-Protocol-Version': '1',
};

function randomSuffix(): string {
  return randomBytes(4).toString('hex');
}

function requireAdminToken(): string {
  const token = process.env.AGYN_API_TOKEN;
  if (!token) {
    throw new Error('AGYN_API_TOKEN env var is required for admin operations in e2e.');
  }
  return token;
}

async function getOidcAccessToken(browser: Browser, email: string): Promise<string> {
  const context = await browser.newContext();
  const page = await context.newPage();
  try {
    await signInViaOidc(page, email, { ensureAdmin: false });
    const session = await readOidcSession(page);
    const token = session?.accessToken;
    if (!token) {
      throw new Error(`OIDC access token missing for ${email}.`);
    }
    return token;
  } finally {
    await context.close();
  }
}

type UserDirectoryEntry = {
  identityId?: string;
  username?: string;
  name?: string;
  photoUrl?: string;
};

type ClusterRoleWire = string | number | undefined;

type GetMeResponseWire = {
  clusterRole?: string | number;
};

type MembershipWire = {
  id: string;
  status?: string | number;
};

type MembershipResponseWire = {
  id?: string;
  status?: string | number;
};

type ThreadParticipantWire = {
  id?: string;
};

type ThreadWire = {
  id?: string;
  participants?: ThreadParticipantWire[];
};

type CreateThreadResponseWire = {
  thread?: ThreadWire;
};

const CLUSTER_ROLE_ADMIN = 1;
const MEMBERSHIP_STATUS_PENDING = 1;

function isClusterAdminRole(role: ClusterRoleWire): boolean {
  if (role === undefined || role === null) return false;
  if (typeof role === 'number') return role === CLUSTER_ROLE_ADMIN;
  return role === 'CLUSTER_ROLE_ADMIN';
}

function isPendingMembershipStatus(status: MembershipWire['status']): boolean {
  if (status === undefined || status === null) return false;
  if (typeof status === 'number') return status === MEMBERSHIP_STATUS_PENDING;
  return status === 'MEMBERSHIP_STATUS_PENDING';
}

async function postConnect<T>(
  request: APIRequestContext,
  path: string,
  method: string,
  data: Record<string, unknown>,
  token: string,
): Promise<T> {
  if (!token) {
    throw new Error('Access token is required for user-directory e2e requests.');
  }
  const headers = {
    ...CONNECT_HEADERS,
    Authorization: `Bearer ${token}`,
  };
  const response = await request.post(`${path}/${method}`, { data, headers });
  if (!response.ok()) {
    const body = await response.text();
    throw new Error(`${method} failed (${response.status()}): ${body}`);
  }
  return (await response.json()) as T;
}

async function createUser(
  request: APIRequestContext,
  token: string,
  opts: { email: string; username: string; name?: string; photoUrl?: string; nickname?: string },
): Promise<string> {
  const response = await postConnect<{ user?: { meta?: { id?: string } } }>(
    request,
    USERS_GATEWAY_PATH,
    'CreateUser',
    {
      username: opts.username,
      oidcSubject: opts.email,
      name: opts.name ?? opts.username,
      nickname: opts.nickname ?? opts.username,
      photoUrl: opts.photoUrl ?? '',
      clusterRole: 'CLUSTER_ROLE_UNSPECIFIED',
    },
    token,
  );
  const identityId = response.user?.meta?.id;
  if (!identityId) {
    throw new Error('CreateUser response missing identity id.');
  }
  return identityId;
}

async function createOrganization(request: APIRequestContext, token: string, name: string): Promise<string> {
  const response = await postConnect<{ organization?: { id?: string } }>(
    request,
    ORGS_GATEWAY_PATH,
    'CreateOrganization',
    { name },
    token,
  );
  const organizationId = response.organization?.id;
  if (!organizationId) {
    throw new Error('CreateOrganization response missing organization id.');
  }
  return organizationId;
}

async function searchUsers(
  request: APIRequestContext,
  token: string,
  opts: { prefix: string },
): Promise<UserDirectoryEntry[]> {
  const response = await postConnect<{ users?: UserDirectoryEntry[] }>(
    request,
    USERS_GATEWAY_PATH,
    'SearchUsers',
    { prefix: opts.prefix, limit: 10 },
    token,
  );
  return response.users ?? [];
}

async function createMembership(
  request: APIRequestContext,
  token: string,
  opts: { organizationId: string; identityId: string },
): Promise<MembershipWire> {
  const response = await postConnect<{ membership?: MembershipResponseWire }>(
    request,
    ORGS_GATEWAY_PATH,
    'CreateMembership',
    {
      organizationId: opts.organizationId,
      identityId: opts.identityId,
      role: 'MEMBERSHIP_ROLE_MEMBER',
    },
    token,
  );
  const membership = response.membership;
  if (!membership?.id) {
    throw new Error('CreateMembership response missing membership id.');
  }
  return { id: membership.id, status: membership.status };
}

async function acceptMembership(
  request: APIRequestContext,
  token: string,
  opts: { membershipId: string },
): Promise<void> {
  await postConnect(
    request,
    ORGS_GATEWAY_PATH,
    'AcceptMembership',
    { membershipId: opts.membershipId },
    token,
  );
}

async function updateUsername(
  request: APIRequestContext,
  token: string,
  opts: { username: string },
): Promise<void> {
  await postConnect(
    request,
    USERS_GATEWAY_PATH,
    'UpdateMe',
    { username: opts.username },
    token,
  );
}

async function getMe(request: APIRequestContext, token: string): Promise<GetMeResponseWire> {
  return postConnect<GetMeResponseWire>(request, USERS_GATEWAY_PATH, 'GetMe', {}, token);
}

async function createThreadByNickname(
  request: APIRequestContext,
  token: string,
  opts: { organizationId: string; nickname: string },
): Promise<ThreadWire> {
  const response = await postConnect<CreateThreadResponseWire>(
    request,
    THREADS_GATEWAY_PATH,
    'CreateThread',
    {
      organizationId: opts.organizationId,
      participants: [{ participantNickname: opts.nickname }],
    },
    token,
  );
  const thread = response.thread;
  if (!thread?.id) {
    throw new Error('CreateThread response missing thread id.');
  }
  return thread;
}

test.describe('user-directory', { tag: ['@svc_console', '@issue140'] }, () => {
  test('non-admin SearchUsers redacts profile fields', async ({ request, browser }) => {
    const adminToken = requireAdminToken();
    const suffix = randomSuffix();
    const targetUsername = `e2e-search-${suffix}`;
    const callerUsername = `e2e-caller-${suffix}`;
    const targetEmail = `${targetUsername}@agyn.test`;
    const callerEmail = `${callerUsername}@agyn.test`;

    const targetId = await createUser(request, adminToken, {
      email: targetEmail,
      username: targetUsername,
      name: 'Redacted User',
      photoUrl: 'https://example.com/photo.png',
    });
    await createUser(request, adminToken, {
      email: callerEmail,
      username: callerUsername,
    });

    const callerToken = await getOidcAccessToken(browser, callerEmail);
    const me = await getMe(request, callerToken);
    expect(isClusterAdminRole(me.clusterRole)).toBe(false);
    const results = await searchUsers(request, callerToken, { prefix: targetUsername });
    expect(results).toHaveLength(1);
    const entry = results[0];
    expect(entry.identityId).toBe(targetId);
    expect(entry.username).toBe(targetUsername);
    expect(entry.name ?? '').toBe('Redacted User');
    expect(entry.photoUrl ?? '').toBe('https://example.com/photo.png');
  });

  test('invite by username seeds org nickname on accept', async ({ request, browser }) => {
    const adminToken = requireAdminToken();
    const suffix = randomSuffix();
    const inviterUsername = `e2e-inviter-${suffix}`;
    const inviteeUsername = `e2e-invite-${suffix}`;
    const inviterEmail = `${inviterUsername}@agyn.test`;
    const inviteeEmail = `${inviteeUsername}@agyn.test`;

    const inviterId = await createUser(request, adminToken, {
      email: inviterEmail,
      username: inviterUsername,
      name: 'Inviter User',
    });
    const inviteeId = await createUser(request, adminToken, {
      email: inviteeEmail,
      username: inviteeUsername,
      name: 'Invitee User',
    });

    const inviterToken = await getOidcAccessToken(browser, inviterEmail);
    const organizationId = await createOrganization(request, inviterToken, `e2e-org-${suffix}`);
    const results = await searchUsers(request, inviterToken, { prefix: inviteeUsername });
    expect(results).toHaveLength(1);
    const inviteeEntry = results[0];
    const entryIdentityId = inviteeEntry?.identityId;
    if (!entryIdentityId) {
      throw new Error('SearchUsers response missing identity id.');
    }
    expect(entryIdentityId).toBe(inviteeId);
    const membership = await createMembership(request, inviterToken, {
      organizationId,
      identityId: entryIdentityId,
    });
    expect(isPendingMembershipStatus(membership.status)).toBe(true);

    const inviteeToken = await getOidcAccessToken(browser, inviteeEmail);
    await acceptMembership(request, inviteeToken, { membershipId: membership.id });

    const thread = await createThreadByNickname(request, inviterToken, {
      organizationId,
      nickname: inviteeUsername,
    });
    expect(thread.participants?.some((participant) => participant.id === inviteeId) ?? false).toBe(true);
    expect(inviterId).not.toBe(inviteeId);
  });

  test('renaming username does not change existing org nickname', async ({ request, browser }) => {
    const adminToken = requireAdminToken();
    const suffix = randomSuffix();
    const inviterUsername = `e2e-inviter-rename-${suffix}`;
    const originalUsername = `e2e-user-${suffix}`;
    const newUsername = `e2e-user-new-${suffix}`;
    const inviterEmail = `${inviterUsername}@agyn.test`;
    const inviteeEmail = `${originalUsername}@agyn.test`;

    const inviterId = await createUser(request, adminToken, {
      email: inviterEmail,
      username: inviterUsername,
      name: 'Inviter User',
    });
    const inviteeId = await createUser(request, adminToken, {
      email: inviteeEmail,
      username: originalUsername,
      name: 'Rename User',
    });

    const inviterToken = await getOidcAccessToken(browser, inviterEmail);
    const organizationId = await createOrganization(request, inviterToken, `e2e-org-rename-${suffix}`);
    const membership = await createMembership(request, inviterToken, {
      organizationId,
      identityId: inviteeId,
    });
    expect(isPendingMembershipStatus(membership.status)).toBe(true);
    const inviteeToken = await getOidcAccessToken(browser, inviteeEmail);
    await acceptMembership(request, inviteeToken, { membershipId: membership.id });

    await updateUsername(request, inviteeToken, { username: newUsername });

    const thread = await createThreadByNickname(request, inviterToken, {
      organizationId,
      nickname: originalUsername,
    });
    expect(thread.participants?.some((participant) => participant.id === inviteeId) ?? false).toBe(true);
    expect(inviterId).not.toBe(inviteeId);
  });
});
