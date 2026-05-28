# Playwright Console App

- **Suite ID:** E2E-SUITE-PLAYWRIGHT
- **Source:** `suites/playwright/suite.yaml`
- **Kind:** Playwright
- **Tags:** @svc_console, @svc_gateway, @svc_threads, @svc_metering, @svc_identity, @smoke

## Intent

Validates Console App user journeys and browser-visible API behaviors for sign-in, organizations, agents, runners, devices, settings, usage, threads, users, and layout.

## Scope

- Source directory: `suites/playwright`
- Test inventory pattern: `test/e2e/*.spec.ts`
- Included case count: 41

## Actors

- Console App user
- Organization administrator
- Gateway API
- OIDC identity provider

## Preconditions

- The Console App is served at `E2E_BASE_URL`.
- OIDC sign-in works for the configured test user.
- The backing platform services and generated ConnectRPC clients are available.

## Tags

- `@svc_console`
- `@svc_gateway`
- `@svc_threads`
- `@svc_metering`
- `@svc_identity`
- `@smoke`

## Case index

| Case ID | Source test | Tags |
| --- | --- | --- |
| [E2E-PLAYWRIGHT-001](#e2e-playwright-001) | `manages MCP image pull secrets dialog` | @svc_console |
| [E2E-PLAYWRIGHT-002](#e2e-playwright-002) | `manages Hook image pull secrets dialog` | @svc_console |
| [E2E-PLAYWRIGHT-003](#e2e-playwright-003) | `shows agent configuration and edit dialog` | @svc_console |
| [E2E-PLAYWRIGHT-004](#e2e-playwright-004) | `shows agent create form` | @svc_console |
| [E2E-PLAYWRIGHT-006](#e2e-playwright-006) | `CreateAgent payload defaults availability to internal` | _none_ |
| [E2E-PLAYWRIGHT-007](#e2e-playwright-007) | `CreateAgent payload keeps caller availability override` | _none_ |
| [E2E-PLAYWRIGHT-008](#e2e-playwright-008) | `CreateAgent ConnectRPC JSON uses protobuf enum name` | _none_ |
| [E2E-PLAYWRIGHT-009](#e2e-playwright-009) | `CreateAgent ConnectRPC proto bytes include availability value` | _none_ |
| [E2E-PLAYWRIGHT-010](#e2e-playwright-010) | `dashboard shows stats cards` | @svc_console |
| [E2E-PLAYWRIGHT-011](#e2e-playwright-011) | `shows empty devices page` | @svc_console |
| [E2E-PLAYWRIGHT-012](#e2e-playwright-012) | `creates a device and shows enrollment JWT` | @svc_console |
| [E2E-PLAYWRIGHT-013](#e2e-playwright-013) | `deletes a device with confirmation` | @svc_console |
| [E2E-PLAYWRIGHT-014](#e2e-playwright-014) | `tests model successfully` | @svc_console |
| [E2E-PLAYWRIGHT-015](#e2e-playwright-015) | `shows model test failure` | @svc_console |
| [E2E-PLAYWRIGHT-016](#e2e-playwright-016) | `shows current user as member` | @svc_console |
| [E2E-PLAYWRIGHT-017](#e2e-playwright-017) | `invite member shows pending entry` | @svc_console |
| [E2E-PLAYWRIGHT-018](#e2e-playwright-018) | `change member role updates list` | @svc_console |
| [E2E-PLAYWRIGHT-019](#e2e-playwright-019) | `remove member clears list entry` | @svc_console |
| [E2E-PLAYWRIGHT-020](#e2e-playwright-020) | `records thread and message usage after platform activity` | @svc_console, @svc_metering |
| [E2E-PLAYWRIGHT-021](#e2e-playwright-021) | `shows empty secrets tab initially` | @svc_console |
| [E2E-PLAYWRIGHT-022](#e2e-playwright-022) | `shows secret providers and secrets` | @svc_console |
| [E2E-PLAYWRIGHT-023](#e2e-playwright-023) | `threads list loads with data` | @svc_console, @svc_gateway, @svc_threads, @svc_identity, @smoke |
| [E2E-PLAYWRIGHT-024](#e2e-playwright-024) | `org threads list and detail pagination` | @svc_console, @svc_gateway, @svc_threads, @svc_identity |
| [E2E-PLAYWRIGHT-025](#e2e-playwright-025) | `shows populated usage dashboard after LLM call` | @svc_console |
| [E2E-PLAYWRIGHT-026](#e2e-playwright-026) | `shows empty state for range with no data` | @svc_console |
| [E2E-PLAYWRIGHT-027](#e2e-playwright-027) | `lists organizations` | @svc_console |
| [E2E-PLAYWRIGHT-028](#e2e-playwright-028) | `org detail shows overview` | @svc_console |
| [E2E-PLAYWRIGHT-029](#e2e-playwright-029) | `lists cluster runners` | @svc_console |
| [E2E-PLAYWRIGHT-030](#e2e-playwright-030) | `lists organization and cluster runners` | @svc_console |
| [E2E-PLAYWRIGHT-031](#e2e-playwright-031) | `organization runner detail shows metadata` | @svc_console |
| [E2E-PLAYWRIGHT-032](#e2e-playwright-032) | `runner detail shows metadata` | @svc_console |
| [E2E-PLAYWRIGHT-033](#e2e-playwright-033) | `shows settings profile info` | @svc_console |
| [E2E-PLAYWRIGHT-034](#e2e-playwright-034) | `signs in via oidc redirect flow` | @svc_console, @smoke |
| [E2E-PLAYWRIGHT-035](#e2e-playwright-035) | `signs out from user menu` | @svc_console |
| [E2E-PLAYWRIGHT-036](#e2e-playwright-036) | `signs out from settings page` | @svc_console |
| [E2E-PLAYWRIGHT-037](#e2e-playwright-037) | `non-admin SearchUsers redacts profile fields` | @svc_console, @issue140 |
| [E2E-PLAYWRIGHT-038](#e2e-playwright-038) | `invite by username seeds org nickname on accept` | @svc_console, @issue140 |
| [E2E-PLAYWRIGHT-039](#e2e-playwright-039) | `renaming username does not change existing org nickname` | @svc_console, @issue140 |
| [E2E-PLAYWRIGHT-040](#e2e-playwright-040) | `lists users` | @svc_console |
| [E2E-PLAYWRIGHT-041](#e2e-playwright-041) | `shows user detail` | @svc_console |
| [E2E-PLAYWRIGHT-042](#e2e-playwright-042) | `workloads header lays out across columns` | @svc_console, @svc_gateway, @smoke |

## Scenarios

### E2E-PLAYWRIGHT-001

- **Source:** `suites/playwright/test/e2e/agent-image-pull-secrets.spec.ts`
- **Test:** `manages MCP image pull secrets dialog`
- **Tags:** @svc_console

**Scenario:** manages MCP image pull secrets dialog

- **Given** An organization has an agent, MCP, and image pull secret.
- **When** The user opens the MCP image-pull-secrets dialog, attaches a secret, and detaches it.
- **Then** The dialog shows empty, attached, and empty-again states for the MCP secret.

### E2E-PLAYWRIGHT-002

- **Source:** `suites/playwright/test/e2e/agent-image-pull-secrets.spec.ts`
- **Test:** `manages Hook image pull secrets dialog`
- **Tags:** @svc_console

**Scenario:** manages Hook image pull secrets dialog

- **Given** An organization has an agent, hook, and image pull secret.
- **When** The user opens the hook image-pull-secrets dialog, attaches a secret, and detaches it.
- **Then** The dialog shows empty, attached, and empty-again states for the hook secret.

### E2E-PLAYWRIGHT-003

- **Source:** `suites/playwright/test/e2e/agents.spec.ts`
- **Test:** `shows agent configuration and edit dialog`
- **Tags:** @svc_console

**Scenario:** shows agent configuration and edit dialog

- **Given** An organization has an agent with configuration.
- **When** The user opens the agent detail page and opens the edit dialog.
- **Then** The configuration card and nickname edit field are visible.

### E2E-PLAYWRIGHT-004

- **Source:** `suites/playwright/test/e2e/agents.spec.ts`
- **Test:** `shows agent create form`
- **Tags:** @svc_console

**Scenario:** shows agent create form

- **Given** The user has selected an organization.
- **When** The user opens the new-agent route.
- **Then** The agent create form and nickname input are visible.

### E2E-PLAYWRIGHT-006

- **Source:** `suites/playwright/test/e2e/console-api.spec.ts`
- **Test:** `CreateAgent payload defaults availability to internal`
- **Tags:** _none_

**Scenario:** CreateAgent payload defaults availability to internal

- **Given** A CreateAgent helper call omits availability.
- **When** The payload is serialized.
- **Then** The payload uses internal availability by default.

### E2E-PLAYWRIGHT-007

- **Source:** `suites/playwright/test/e2e/console-api.spec.ts`
- **Test:** `CreateAgent payload keeps caller availability override`
- **Tags:** _none_

**Scenario:** CreateAgent payload keeps caller availability override

- **Given** A CreateAgent helper call includes a private availability override.
- **When** The payload is serialized.
- **Then** The payload preserves the caller-provided availability.

### E2E-PLAYWRIGHT-008

- **Source:** `suites/playwright/test/e2e/console-api.spec.ts`
- **Test:** `CreateAgent ConnectRPC JSON uses protobuf enum name`
- **Tags:** _none_

**Scenario:** CreateAgent ConnectRPC JSON uses protobuf enum name

- **Given** A CreateAgent request is built for ConnectRPC JSON.
- **When** The JSON payload is serialized.
- **Then** The availability enum is represented by the protobuf enum name.

### E2E-PLAYWRIGHT-009

- **Source:** `suites/playwright/test/e2e/console-api.spec.ts`
- **Test:** `CreateAgent ConnectRPC proto bytes include availability value`
- **Tags:** _none_

**Scenario:** CreateAgent ConnectRPC proto bytes include availability value

- **Given** A CreateAgent request is built for protobuf bytes.
- **When** The bytes are serialized.
- **Then** The encoded bytes include the availability value.

### E2E-PLAYWRIGHT-010

- **Source:** `suites/playwright/test/e2e/dashboard.spec.ts`
- **Test:** `dashboard shows stats cards`
- **Tags:** @svc_console

**Scenario:** dashboard shows stats cards

- **Given** The Console App user is signed in.
- **When** The user opens the dashboard.
- **Then** The dashboard title and three stats cards are visible.

### E2E-PLAYWRIGHT-011

- **Source:** `suites/playwright/test/e2e/devices.spec.ts`
- **Test:** `shows empty devices page`
- **Tags:** @svc_console

**Scenario:** shows empty devices page

- **Given** No E2E-created devices exist.
- **When** The user opens the devices page.
- **Then** The page shows the empty devices state.

### E2E-PLAYWRIGHT-012

- **Source:** `suites/playwright/test/e2e/devices.spec.ts`
- **Test:** `creates a device and shows enrollment JWT`
- **Tags:** @svc_console

**Scenario:** creates a device and shows enrollment JWT

- **Given** The user can create devices.
- **When** The user creates a device.
- **Then** The UI displays an enrollment JWT and then lists the created device.

### E2E-PLAYWRIGHT-013

- **Source:** `suites/playwright/test/e2e/devices.spec.ts`
- **Test:** `deletes a device with confirmation`
- **Tags:** @svc_console

**Scenario:** deletes a device with confirmation

- **Given** A device exists.
- **When** The user confirms deletion from the devices list.
- **Then** The device row is removed from the list.

### E2E-PLAYWRIGHT-014

- **Source:** `suites/playwright/test/e2e/model-test.spec.ts`
- **Test:** `tests model successfully`
- **Tags:** @svc_console

**Scenario:** tests model successfully

- **Given** A model test form is available.
- **When** The user tests a working model configuration.
- **Then** The UI reports the model test as successful. TODO: clarify.

### E2E-PLAYWRIGHT-015

- **Source:** `suites/playwright/test/e2e/model-test.spec.ts`
- **Test:** `shows model test failure`
- **Tags:** @svc_console

**Scenario:** shows model test failure

- **Given** A model test form is available.
- **When** The user tests a failing model configuration.
- **Then** The UI reports the model test failure. TODO: clarify.

### E2E-PLAYWRIGHT-016

- **Source:** `suites/playwright/test/e2e/organization-members.spec.ts`
- **Test:** `shows current user as member`
- **Tags:** @svc_console

**Scenario:** shows current user as member

- **Given** The current user belongs to an organization.
- **When** The user opens organization members.
- **Then** The members list shows the current user.

### E2E-PLAYWRIGHT-017

- **Source:** `suites/playwright/test/e2e/organization-members.spec.ts`
- **Test:** `invite member shows pending entry`
- **Tags:** @svc_console

**Scenario:** invite member shows pending entry

- **Given** The user can invite organization members.
- **When** The user invites a member.
- **Then** The members list shows a pending invitation entry.

### E2E-PLAYWRIGHT-018

- **Source:** `suites/playwright/test/e2e/organization-members.spec.ts`
- **Test:** `change member role updates list`
- **Tags:** @svc_console

**Scenario:** change member role updates list

- **Given** An organization member exists.
- **When** The user changes the member role.
- **Then** The members list shows the updated role.

### E2E-PLAYWRIGHT-019

- **Source:** `suites/playwright/test/e2e/organization-members.spec.ts`
- **Test:** `remove member clears list entry`
- **Tags:** @svc_console

**Scenario:** remove member clears list entry

- **Given** An organization member exists.
- **When** The user removes the member.
- **Then** The member entry disappears from the list.

### E2E-PLAYWRIGHT-020

- **Source:** `suites/playwright/test/e2e/organization-platform-usage.spec.ts`
- **Test:** `records thread and message usage after platform activity`
- **Tags:** @svc_console, @svc_metering

**Scenario:** records thread and message usage after platform activity

- **Given** Metering is enabled and platform thread/message activity can be generated.
- **When** The test creates platform activity and opens usage reporting.
- **Then** Thread and message usage appears in the organization platform usage dashboard.

### E2E-PLAYWRIGHT-021

- **Source:** `suites/playwright/test/e2e/organization-secrets.spec.ts`
- **Test:** `shows empty secrets tab initially`
- **Tags:** @svc_console

**Scenario:** shows empty secrets tab initially

- **Given** An organization has no configured secrets.
- **When** The user opens the organization secrets tab.
- **Then** The UI shows the empty secrets state.

### E2E-PLAYWRIGHT-022

- **Source:** `suites/playwright/test/e2e/organization-secrets.spec.ts`
- **Test:** `shows secret providers and secrets`
- **Tags:** @svc_console

**Scenario:** shows secret providers and secrets

- **Given** Secret providers and secrets are configured.
- **When** The user opens the organization secrets tab.
- **Then** The UI lists the available providers and secrets.

### E2E-PLAYWRIGHT-023

- **Source:** `suites/playwright/test/e2e/organization-threads-smoke.spec.ts`
- **Test:** `threads list loads with data`
- **Tags:** @svc_console, @svc_gateway, @svc_threads, @svc_identity, @smoke

**Scenario:** threads list loads with data

- **Given** An organization has thread data.
- **When** The user opens the threads list.
- **Then** The threads list loads with at least one row.

### E2E-PLAYWRIGHT-024

- **Source:** `suites/playwright/test/e2e/organization-threads.spec.ts`
- **Test:** `org threads list and detail pagination`
- **Tags:** @svc_console, @svc_gateway, @svc_threads, @svc_identity

**Scenario:** org threads list and detail pagination

- **Given** An organization has enough threads and messages for pagination.
- **When** The user opens the thread list and a thread detail view.
- **Then** List and detail pagination controls reveal additional data correctly.

### E2E-PLAYWRIGHT-025

- **Source:** `suites/playwright/test/e2e/organization-usage.spec.ts`
- **Test:** `shows populated usage dashboard after LLM call`
- **Tags:** @svc_console

**Scenario:** shows populated usage dashboard after LLM call

- **Given** An LLM call can be made for an organization.
- **When** The user opens the usage dashboard after usage is recorded.
- **Then** The dashboard shows populated usage metrics.

### E2E-PLAYWRIGHT-026

- **Source:** `suites/playwright/test/e2e/organization-usage.spec.ts`
- **Test:** `shows empty state for range with no data`
- **Tags:** @svc_console

**Scenario:** shows empty state for range with no data

- **Given** A date range has no usage data.
- **When** The user selects that range on the usage dashboard.
- **Then** The dashboard shows an empty state.

### E2E-PLAYWRIGHT-027

- **Source:** `suites/playwright/test/e2e/organizations.spec.ts`
- **Test:** `lists organizations`
- **Tags:** @svc_console

**Scenario:** lists organizations

- **Given** The user belongs to organizations.
- **When** The user opens the organizations list.
- **Then** The organizations list is visible.

### E2E-PLAYWRIGHT-028

- **Source:** `suites/playwright/test/e2e/organizations.spec.ts`
- **Test:** `org detail shows overview`
- **Tags:** @svc_console

**Scenario:** org detail shows overview

- **Given** An organization exists.
- **When** The user opens organization detail.
- **Then** The overview card or section is visible.

### E2E-PLAYWRIGHT-029

- **Source:** `suites/playwright/test/e2e/runners.spec.ts`
- **Test:** `lists cluster runners`
- **Tags:** @svc_console

**Scenario:** lists cluster runners

- **Given** Cluster runners are available.
- **When** The user opens the runners list.
- **Then** Cluster runners are listed.

### E2E-PLAYWRIGHT-030

- **Source:** `suites/playwright/test/e2e/runners.spec.ts`
- **Test:** `lists organization and cluster runners`
- **Tags:** @svc_console

**Scenario:** lists organization and cluster runners

- **Given** Organization and cluster runners are available.
- **When** The user opens runner views for an organization.
- **Then** Both organization and cluster runners are listed.

### E2E-PLAYWRIGHT-031

- **Source:** `suites/playwright/test/e2e/runners.spec.ts`
- **Test:** `organization runner detail shows metadata`
- **Tags:** @svc_console

**Scenario:** organization runner detail shows metadata

- **Given** An organization runner exists.
- **When** The user opens organization runner detail.
- **Then** Runner metadata is visible.

### E2E-PLAYWRIGHT-032

- **Source:** `suites/playwright/test/e2e/runners.spec.ts`
- **Test:** `runner detail shows metadata`
- **Tags:** @svc_console

**Scenario:** runner detail shows metadata

- **Given** A runner exists.
- **When** The user opens runner detail.
- **Then** Runner metadata is visible.

### E2E-PLAYWRIGHT-033

- **Source:** `suites/playwright/test/e2e/settings.spec.ts`
- **Test:** `shows settings profile info`
- **Tags:** @svc_console

**Scenario:** shows settings profile info

- **Given** The user is signed in.
- **When** The user opens settings.
- **Then** The profile information is visible.

### E2E-PLAYWRIGHT-034

- **Source:** `suites/playwright/test/e2e/sign-in.spec.ts`
- **Test:** `signs in via oidc redirect flow`
- **Tags:** @svc_console, @smoke

**Scenario:** signs in via oidc redirect flow

- **Given** The user is not signed in.
- **When** The user completes the OIDC redirect flow.
- **Then** The Console App shows an authenticated session.

### E2E-PLAYWRIGHT-035

- **Source:** `suites/playwright/test/e2e/sign-out.spec.ts`
- **Test:** `signs out from user menu`
- **Tags:** @svc_console

**Scenario:** signs out from user menu

- **Given** The user is signed in.
- **When** The user signs out from the user menu.
- **Then** The session ends and the app returns to an unauthenticated state.

### E2E-PLAYWRIGHT-036

- **Source:** `suites/playwright/test/e2e/sign-out.spec.ts`
- **Test:** `signs out from settings page`
- **Tags:** @svc_console

**Scenario:** signs out from settings page

- **Given** The user is signed in and viewing settings.
- **When** The user signs out from the settings page.
- **Then** The session ends and the app returns to an unauthenticated state.

### E2E-PLAYWRIGHT-037

- **Source:** `suites/playwright/test/e2e/user-directory-api.spec.ts`
- **Test:** `non-admin SearchUsers redacts profile fields`
- **Tags:** @svc_console, @issue140

**Scenario:** non-admin SearchUsers redacts profile fields

- **Given** A non-admin user searches the user directory.
- **When** SearchUsers returns matching users.
- **Then** Sensitive profile fields are redacted for the non-admin caller.

### E2E-PLAYWRIGHT-038

- **Source:** `suites/playwright/test/e2e/user-directory-api.spec.ts`
- **Test:** `invite by username seeds org nickname on accept`
- **Tags:** @svc_console, @issue140

**Scenario:** invite by username seeds org nickname on accept

- **Given** A user is invited to an organization by username.
- **When** The invitee accepts the membership.
- **Then** The organization nickname is seeded from the username.

### E2E-PLAYWRIGHT-039

- **Source:** `suites/playwright/test/e2e/user-directory-api.spec.ts`
- **Test:** `renaming username does not change existing org nickname`
- **Tags:** @svc_console, @issue140

**Scenario:** renaming username does not change existing org nickname

- **Given** A user already has an organization nickname seeded from username.
- **When** The user renames their username.
- **Then** The existing organization nickname remains unchanged.

### E2E-PLAYWRIGHT-040

- **Source:** `suites/playwright/test/e2e/users.spec.ts`
- **Test:** `lists users`
- **Tags:** @svc_console

**Scenario:** lists users

- **Given** Users exist in the platform.
- **When** The user opens the users list.
- **Then** The list includes the current user.

### E2E-PLAYWRIGHT-041

- **Source:** `suites/playwright/test/e2e/users.spec.ts`
- **Test:** `shows user detail`
- **Tags:** @svc_console

**Scenario:** shows user detail

- **Given** The current user exists.
- **When** The user opens their user detail page.
- **Then** The user profile card is visible.

### E2E-PLAYWRIGHT-042

- **Source:** `suites/playwright/test/e2e/workloads-layout.spec.ts`
- **Test:** `workloads header lays out across columns`
- **Tags:** @svc_console, @svc_gateway, @smoke

**Scenario:** workloads header lays out across columns

- **Given** An organization activity workloads page is available.
- **When** The user opens the workloads page.
- **Then** The Agent and Status headers are visible and aligned in separate columns.
