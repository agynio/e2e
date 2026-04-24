import { randomUUID } from 'node:crypto';
import { expect, test, type APIRequestContext } from '@playwright/test';

const USERS_GATEWAY_PATH = '/api/agynio.api.gateway.v1.UsersGateway';
const ORGS_GATEWAY_PATH = '/api/agynio.api.gateway.v1.OrganizationsGateway';
const IDENTITY_SERVICE_PATH = '/api/agynio.api.identity.v1.IdentityService';

const CONNECT_HEADERS = {
  'Content-Type': 'application/json',
  'Connect-Protocol-Version': '1',
};

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
  identityId?: string,
): Promise<T> {
  const headers = {
    ...CONNECT_HEADERS,
    ...(identityId ? { 'x-e2e-identity-id': identityId } : {}),
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
  opts: { email: string; username: string; name?: string; photoUrl?: string; nickname?: string },
): Promise<string> {
  const response = await postConnect<{ user?: { meta?: { id?: string } } }>(
    request,
    USERS_GATEWAY_PATH,
    'CreateUser',
    {
      email: opts.email,
      username: opts.username,
      name: opts.name ?? opts.username,
      nickname: opts.nickname ?? opts.username,
      photoUrl: opts.photoUrl ?? '',
    },
  );
  const identityId = response.user?.meta?.id;
  if (!identityId) {
    throw new Error('CreateUser response missing identity id.');
  }
  return identityId;
}

async function createOrganization(request: APIRequestContext, name: string): Promise<string> {
  const response = await postConnect<{ organization?: { id?: string } }>(
    request,
    ORGS_GATEWAY_PATH,
    'CreateOrganization',
    { name },
  );
  const organizationId = response.organization?.id;
  if (!organizationId) {
    throw new Error('CreateOrganization response missing organization id.');
  }
  return organizationId;
}

async function searchUsers(
  request: APIRequestContext,
  opts: { prefix: string; identityId?: string },
): Promise<UserDirectoryEntry[]> {
  const response = await postConnect<{ users?: UserDirectoryEntry[] }>(
    request,
    USERS_GATEWAY_PATH,
    'SearchUsers',
    { prefix: opts.prefix, limit: 10 },
    opts.identityId,
  );
  return response.users ?? [];
}

async function createMembership(
  request: APIRequestContext,
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
  );
  const membershipId = response.membership?.id;
  if (!membershipId) {
    throw new Error('CreateMembership response missing membership id.');
  }
  return membershipId;
}

async function acceptMembership(
  request: APIRequestContext,
  opts: { membershipId: string; identityId: string },
): Promise<void> {
  await postConnect(
    request,
    ORGS_GATEWAY_PATH,
    'AcceptMembership',
    { membershipId: opts.membershipId },
    opts.identityId,
  );
}

async function updateUsername(
  request: APIRequestContext,
  opts: { identityId: string; username: string },
): Promise<void> {
  await postConnect(
    request,
    USERS_GATEWAY_PATH,
    'UpdateMe',
    { username: opts.username },
    opts.identityId,
  );
}

async function resolveOrgNickname(
  request: APIRequestContext,
  opts: { organizationId: string; nickname: string },
): Promise<string> {
  const response = await postConnect<{ identityId?: string }>(
    request,
    IDENTITY_SERVICE_PATH,
    'ResolveNickname',
    { organizationId: opts.organizationId, nickname: opts.nickname },
  );
  const identityId = response.identityId;
  if (!identityId) {
    throw new Error('ResolveNickname response missing identity id.');
  }
  return identityId;
}

test.describe('user-directory', { tag: ['@svc_console', '@issue140'] }, () => {
  test('non-admin SearchUsers redacts profile fields', async ({ request }) => {
    const suffix = randomUUID();
    const targetUsername = `e2e-search-${suffix}`;
    const callerUsername = `e2e-caller-${suffix}`;

    const targetId = await createUser(request, {
      email: `${targetUsername}@agyn.test`,
      username: targetUsername,
      name: 'Redacted User',
      photoUrl: 'https://example.com/photo.png',
    });
    const callerId = await createUser(request, {
      email: `${callerUsername}@agyn.test`,
      username: callerUsername,
    });

    const results = await searchUsers(request, { prefix: targetUsername, identityId: callerId });
    expect(results).toHaveLength(1);
    const entry = results[0];
    expect(entry.identityId).toBe(targetId);
    expect(entry.username).toBe(targetUsername);
    expect(entry.name ?? '').toBe('');
    expect(entry.photoUrl ?? '').toBe('');
  });

  test('invite by username seeds org nickname on accept', async ({ request }) => {
    const suffix = randomUUID();
    const organizationId = await createOrganization(request, `e2e-org-${suffix}`);
    const inviteeUsername = `e2e-invite-${suffix}`;

    const inviteeId = await createUser(request, {
      email: `${inviteeUsername}@agyn.test`,
      username: inviteeUsername,
      name: 'Invitee User',
    });

    const results = await searchUsers(request, { prefix: inviteeUsername });
    expect(results).toHaveLength(1);
    const inviteeEntry = results[0];
    const entryIdentityId = inviteeEntry?.identityId;
    if (!entryIdentityId) {
      throw new Error('SearchUsers response missing identity id.');
    }
    const membershipId = await createMembership(request, {
      organizationId,
      identityId: entryIdentityId,
    });

    await acceptMembership(request, { membershipId, identityId: inviteeId });

    const resolvedId = await resolveOrgNickname(request, {
      organizationId,
      nickname: inviteeUsername,
    });
    expect(resolvedId).toBe(inviteeId);
  });

  test('renaming username does not change existing org nickname', async ({ request }) => {
    const suffix = randomUUID();
    const organizationId = await createOrganization(request, `e2e-org-rename-${suffix}`);
    const originalUsername = `e2e-user-${suffix}`;
    const newUsername = `e2e-user-new-${suffix}`;

    const inviteeId = await createUser(request, {
      email: `${originalUsername}@agyn.test`,
      username: originalUsername,
      name: 'Rename User',
    });

    const membershipId = await createMembership(request, {
      organizationId,
      identityId: inviteeId,
    });
    await acceptMembership(request, { membershipId, identityId: inviteeId });

    await updateUsername(request, { identityId: inviteeId, username: newUsername });

    const resolvedId = await resolveOrgNickname(request, {
      organizationId,
      nickname: originalUsername,
    });
    expect(resolvedId).toBe(inviteeId);
  });
});
