import { randomBytes } from 'node:crypto';
import { expect, test, type APIRequestContext, type Page } from '@playwright/test';
import { ensureMockAuthEmailStrategy, seedOidcSessionViaMockAuth } from './sign-in-helper';
import { readOidcSession } from './oidc-helpers';

const USERS_GATEWAY_PATH = '/api/agynio.api.gateway.v1.UsersGateway';
const ORGS_GATEWAY_PATH = '/api/agynio.api.gateway.v1.OrganizationsGateway';
const IDENTITY_SERVICE_PATH = '/api/agynio.api.identity.v1.IdentityService';

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

async function getOidcAccessToken(page: Page, email: string): Promise<string> {
  await seedOidcSessionViaMockAuth(page, { email, force: true });
  const session = await readOidcSession(page);
  const token = session?.accessToken;
  if (!token) {
    throw new Error(`OIDC access token missing for ${email}.`);
  }
  return token;
}

type UserDirectoryEntry = {
  identityId?: string;
  username?: string;
  name?: string;
  photoUrl?: string;
};

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
      email: opts.email,
      username: opts.username,
      oidcSubject: opts.email,
      name: opts.name ?? opts.username,
      nickname: opts.nickname ?? opts.username,
      photoUrl: opts.photoUrl ?? '',
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
): Promise<string> {
  const response = await postConnect<{ membership?: { id?: string } }>(
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
  const membershipId = response.membership?.id;
  if (!membershipId) {
    throw new Error('CreateMembership response missing membership id.');
  }
  return membershipId;
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

async function resolveOrgNickname(
  request: APIRequestContext,
  token: string,
  opts: { organizationId: string; nickname: string },
): Promise<string> {
  const response = await postConnect<{ identityId?: string }>(
    request,
    IDENTITY_SERVICE_PATH,
    'ResolveNickname',
    { organizationId: opts.organizationId, nickname: opts.nickname },
    token,
  );
  const identityId = response.identityId;
  if (!identityId) {
    throw new Error('ResolveNickname response missing identity id.');
  }
  return identityId;
}

test.describe('user-directory', { tag: ['@svc_console', '@issue140'] }, () => {
  test.beforeEach(async ({ request }) => {
    await ensureMockAuthEmailStrategy(request);
  });

  test('non-admin SearchUsers redacts profile fields', async ({ request, page }) => {
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

    const callerToken = await getOidcAccessToken(page, callerEmail);
    const results = await searchUsers(request, callerToken, { prefix: targetUsername });
    expect(results).toHaveLength(1);
    const entry = results[0];
    expect(entry.identityId).toBe(targetId);
    expect(entry.username).toBe(targetUsername);
    expect(entry.name ?? '').toBe('');
    expect(entry.photoUrl ?? '').toBe('');
  });

  test('invite by username seeds org nickname on accept', async ({ request, page }) => {
    const adminToken = requireAdminToken();
    const suffix = randomSuffix();
    const organizationId = await createOrganization(request, adminToken, `e2e-org-${suffix}`);
    const inviteeUsername = `e2e-invite-${suffix}`;
    const inviteeEmail = `${inviteeUsername}@agyn.test`;

    const inviteeId = await createUser(request, adminToken, {
      email: inviteeEmail,
      username: inviteeUsername,
      name: 'Invitee User',
    });

    const results = await searchUsers(request, adminToken, { prefix: inviteeUsername });
    expect(results).toHaveLength(1);
    const inviteeEntry = results[0];
    const entryIdentityId = inviteeEntry?.identityId;
    if (!entryIdentityId) {
      throw new Error('SearchUsers response missing identity id.');
    }
    expect(entryIdentityId).toBe(inviteeId);
    const membershipId = await createMembership(request, adminToken, {
      organizationId,
      identityId: entryIdentityId,
    });

    const inviteeToken = await getOidcAccessToken(page, inviteeEmail);
    await acceptMembership(request, inviteeToken, { membershipId });

    const resolvedId = await resolveOrgNickname(request, adminToken, {
      organizationId,
      nickname: inviteeUsername,
    });
    expect(resolvedId).toBe(inviteeId);
  });

  test('renaming username does not change existing org nickname', async ({ request, page }) => {
    const adminToken = requireAdminToken();
    const suffix = randomSuffix();
    const organizationId = await createOrganization(request, adminToken, `e2e-org-rename-${suffix}`);
    const originalUsername = `e2e-user-${suffix}`;
    const newUsername = `e2e-user-new-${suffix}`;
    const inviteeEmail = `${originalUsername}@agyn.test`;

    const inviteeId = await createUser(request, adminToken, {
      email: inviteeEmail,
      username: originalUsername,
      name: 'Rename User',
    });

    const membershipId = await createMembership(request, adminToken, {
      organizationId,
      identityId: inviteeId,
    });
    const inviteeToken = await getOidcAccessToken(page, inviteeEmail);
    await acceptMembership(request, inviteeToken, { membershipId });

    await updateUsername(request, inviteeToken, { username: newUsername });

    const resolvedId = await resolveOrgNickname(request, adminToken, {
      organizationId,
      nickname: originalUsername,
    });
    expect(resolvedId).toBe(inviteeId);
  });
});
