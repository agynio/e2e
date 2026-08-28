import { test, expect } from './fixtures';
import { createImage, createOrganization, setSelectedOrganization } from './console-api';

function suffix(): string {
  return `${Date.now()}`;
}

// A repository the platform itself publishes, so the versions the picker offers
// come from real discovery rather than a fixture.
const PUBLIC_REPOSITORY = process.env.TEST_PUBLIC_REPOSITORY ?? 'ghcr.io/agynio/devcontainer-go';

test.describe('organization-images', { tag: ['@svc_console'] }, () => {
  test('registers an image and lists it', async ({ page }) => {
    const now = suffix();
    const organizationId = await createOrganization(page, `e2e-org-images-${now}`);
    await setSelectedOrganization(page, organizationId);

    await page.goto(`/organizations/${organizationId}/images`);
    await expect(page.getByTestId('organization-images')).toBeVisible();
    await expect(page.getByTestId('images-empty')).toBeVisible();

    const name = `e2e-image-${now}`;
    await page.getByTestId('images-register-open').click();
    await page.getByTestId('images-register-name').fill(name);
    await page.getByTestId('images-register-repository').fill(PUBLIC_REPOSITORY);
    await page.getByTestId('images-register-submit').click();

    await expect(page.getByTestId(`image-row-${name}`)).toBeVisible();
    // The repository is what a reader checks the registration against.
    await expect(page.getByTestId(`image-row-${name}`)).toContainText(PUBLIC_REPOSITORY);
  });

  // Registration validates the repository is readable, so a wrong one fails at
  // the dialog rather than at workload start.
  test('refuses a repository it cannot read', async ({ page }) => {
    const now = suffix();
    const organizationId = await createOrganization(page, `e2e-org-images-bad-${now}`);
    await setSelectedOrganization(page, organizationId);

    await page.goto(`/organizations/${organizationId}/images`);
    await page.getByTestId('images-register-open').click();
    await page.getByTestId('images-register-name').fill(`e2e-image-bad-${now}`);
    await page.getByTestId('images-register-repository').fill('ghcr.io/agynio/there-is-no-such-repository');
    await page.getByTestId('images-register-submit').click();

    await expect(page.getByTestId('images-register-name')).toBeVisible();
    await expect(page.getByTestId(`image-row-e2e-image-bad-${now}`)).toHaveCount(0);
  });

  // The point of the catalog: an environment names a registered image and a
  // discovered version instead of a typed reference.
  test('creates an environment from a catalog image and a discovered version', async ({ page }) => {
    const now = suffix();
    const organizationId = await createOrganization(page, `e2e-org-env-images-${now}`);
    await setSelectedOrganization(page, organizationId);

    const name = `e2e-workspace-${now}`;
    await createImage(page, {
      organizationId,
      name,
      repository: PUBLIC_REPOSITORY,
      type: 'IMAGE_TYPE_WORKSPACE',
    });

    await page.goto(`/organizations/${organizationId}/environments`);
    await page.getByTestId('organization-environments-create').click();
    await expect(page.getByTestId('organization-environments-create-dialog')).toBeVisible();

    await page.getByTestId('organization-environments-create-name').fill(`e2e-env-${now}`);

    // A combobox, not a menu: the testid is on a text field and the options are
    // what the typing matches. Clicking alone leaves the list unfiltered and,
    // in an organization with a catalog behind it, not showing this image at
    // all.
    const workspaceImage = page.getByTestId('organization-environments-create-workspace-image');
    await workspaceImage.click();
    // Typing the whole name is the selection: the field resolves a complete
    // name to the image behind it, and the option list closes as it does. An
    // option to click afterwards is not there to be clicked.
    await workspaceImage.fill(name);
    await expect(workspaceImage).toHaveValue(name);

    // The newest version is preselected, so the common case needs no choice.
    const version = page.getByTestId('organization-environments-create-workspace-version');
    await expect(version).toBeVisible();
    await expect(version).not.toHaveText(/Select a version/);

    await page.getByTestId('organization-environments-create-runner').click();
    await page.getByRole('option').first().click();

    await page.getByTestId('organization-environments-create-submit').click();
    // The dialog closing is what says the form was accepted; a required field
    // it is still missing keeps it open and says so in place, which is worth
    // reading rather than waiting out a row that was never going to appear.
    await expect(page.getByTestId('organization-environments-create-dialog')).toBeHidden({ timeout: 15000 });
    await expect(
      page.getByTestId('organization-environment-row').filter({ hasText: `e2e-env-${now}` }),
    ).toBeVisible({ timeout: 15000 });
  });

  test('offers no free-form image field on an environment', async ({ page }) => {
    const now = suffix();
    const organizationId = await createOrganization(page, `e2e-org-noimage-${now}`);
    await setSelectedOrganization(page, organizationId);

    await page.goto(`/organizations/${organizationId}/environments`);
    await page.getByTestId('organization-environments-create').click();
    await expect(page.getByTestId('organization-environments-create-dialog')).toBeVisible();
    await expect(page.getByTestId('organization-environments-create-image')).toHaveCount(0);
  });
});
