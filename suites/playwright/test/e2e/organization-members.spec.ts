import { argosScreenshot } from '@argos-ci/playwright';
import { test, expect } from './fixtures';
import { changeMemberRole, createOrganization, getMe, inviteMember, removeMember, setSelectedOrganization } from './console-api';

test.describe('organization-members', { tag: ['@svc_console'] }, () => {
  test('shows current user as member', async ({ page }) => {
    const organizationId = await createOrganization(page, `e2e-org-members-${Date.now()}`);
    await setSelectedOrganization(page, organizationId);
    const me = await getMe(page);
    const memberLabel = me.user?.name || me.user?.meta?.id;
    if (!memberLabel) {
      throw new Error('GetMe response missing member label for members test.');
    }

    await page.goto(`/organizations/${organizationId}/members`);
    const memberRow = page.getByTestId('organization-member-row').filter({ hasText: memberLabel });
    await expect(memberRow).toBeVisible({ timeout: 15000 });
    await argosScreenshot(page, 'organization-members-list');
  });

  test('invite member shows pending entry', async ({ page }) => {
    const organizationId = await createOrganization(page, `e2e-org-invite-${Date.now()}`);
    await setSelectedOrganization(page, organizationId);
    const inviteEmail = `e2e-member-${Date.now()}@agyn.test`;
    await inviteMember(page, {
      organizationId,
      email: inviteEmail,
      role: 'MEMBERSHIP_ROLE_MEMBER',
    });

    await page.goto(`/organizations/${organizationId}/members`);
    const memberRow = page.getByTestId('organization-member-row').filter({ hasText: inviteEmail });
    await expect(memberRow).toBeVisible({ timeout: 15000 });
    await argosScreenshot(page, 'organization-member-invited');
  });

  test('change member role updates list', async ({ page }) => {
    const organizationId = await createOrganization(page, `e2e-org-role-${Date.now()}`);
    await setSelectedOrganization(page, organizationId);
    const inviteEmail = `e2e-role-${Date.now()}@agyn.test`;
    const invited = await inviteMember(page, {
      organizationId,
      email: inviteEmail,
      role: 'MEMBERSHIP_ROLE_MEMBER',
    });

    await page.goto(`/organizations/${organizationId}/members`);
    const memberRow = page.getByTestId('organization-member-row').filter({ hasText: inviteEmail });
    await expect(memberRow).toBeVisible({ timeout: 15000 });
    await expect(memberRow).toContainText('Member');

    await changeMemberRole(page, {
      organizationId,
      identityId: invited.identityId,
      role: 'MEMBERSHIP_ROLE_OWNER',
    });

    await page.reload();
    const updatedRow = page.getByTestId('organization-member-row').filter({ hasText: inviteEmail });
    await expect(updatedRow).toBeVisible({ timeout: 15000 });
    await expect(updatedRow).toContainText('Owner');
  });

  test('remove member clears list entry', async ({ page }) => {
    const organizationId = await createOrganization(page, `e2e-org-remove-${Date.now()}`);
    await setSelectedOrganization(page, organizationId);
    const inviteEmail = `e2e-remove-${Date.now()}@agyn.test`;
    const invited = await inviteMember(page, {
      organizationId,
      email: inviteEmail,
      role: 'MEMBERSHIP_ROLE_MEMBER',
    });

    await page.goto(`/organizations/${organizationId}/members`);
    const memberRow = page.getByTestId('organization-member-row').filter({ hasText: inviteEmail });
    await expect(memberRow).toBeVisible({ timeout: 15000 });

    await removeMember(page, { organizationId, identityId: invited.identityId });
    await page.reload();
    await expect(page.getByTestId('organization-member-row').filter({ hasText: inviteEmail })).toHaveCount(0);
  });
});
