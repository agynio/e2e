# Go Terraform Provider

- **Suite ID:** E2E-SUITE-GO-TERRAFORM
- **Source:** `suites/go-terraform/suite.yaml`
- **Kind:** Go Terraform acceptance
- **Tags:** @svc_gateway, @tf_provider_agyn

## Intent

Runs Terraform provider acceptance tests against Agyn resources, covering create, update, import, replacement, and validation-error behavior for provider-managed platform resources.

## Scope

- Source directory: `suites/go-terraform`
- Test inventory pattern: `tests/*_test.go`
- Included case count: 47

## Actors

- Terraform user
- Agyn Terraform provider
- Gateway API

## Preconditions

- Terraform acceptance mode is enabled with `TF_ACC=1`.
- Provider credentials and required organization, app, runner, and user inputs are configured.
- The suite can download or use the configured Terraform binary.

## Tags

- `@svc_gateway`
- `@tf_provider_agyn`

## Case index

| Case ID | Source test | Tags |
| --- | --- | --- |
| [E2E-GO-TERRAFORM-001](#e2e-go-terraform-001) | `TestAccAgynAgent_basic` | @svc_gateway, @tf_provider_agyn |
| [E2E-GO-TERRAFORM-002](#e2e-go-terraform-002) | `TestAccAgynAgent_update` | @svc_gateway, @tf_provider_agyn |
| [E2E-GO-TERRAFORM-003](#e2e-go-terraform-003) | `TestAccAgynAgent_import` | @svc_gateway, @tf_provider_agyn |
| [E2E-GO-TERRAFORM-004](#e2e-go-terraform-004) | `TestAccAgynAgent_expectError` | @svc_gateway, @tf_provider_agyn |
| [E2E-GO-TERRAFORM-005](#e2e-go-terraform-005) | `TestAccAgynAgent_invalidNickname` | @svc_gateway, @tf_provider_agyn |
| [E2E-GO-TERRAFORM-006](#e2e-go-terraform-006) | `TestAccAgynAppInstallation_basic` | @svc_gateway, @tf_provider_agyn |
| [E2E-GO-TERRAFORM-007](#e2e-go-terraform-007) | `TestAccAgynAppInstallation_update` | @svc_gateway, @tf_provider_agyn |
| [E2E-GO-TERRAFORM-008](#e2e-go-terraform-008) | `TestAccAgynAppInstallation_import` | @svc_gateway, @tf_provider_agyn |
| [E2E-GO-TERRAFORM-009](#e2e-go-terraform-009) | `TestAccAgynApp_basic` | @svc_gateway, @tf_provider_agyn |
| [E2E-GO-TERRAFORM-010](#e2e-go-terraform-010) | `TestAccAgynApp_import` | @svc_gateway, @tf_provider_agyn |
| [E2E-GO-TERRAFORM-011](#e2e-go-terraform-011) | `TestAccAgynEnv_basic` | @svc_gateway, @tf_provider_agyn |
| [E2E-GO-TERRAFORM-012](#e2e-go-terraform-012) | `TestAccAgynEnv_update` | @svc_gateway, @tf_provider_agyn |
| [E2E-GO-TERRAFORM-013](#e2e-go-terraform-013) | `TestAccAgynEnv_import` | @svc_gateway, @tf_provider_agyn |
| [E2E-GO-TERRAFORM-014](#e2e-go-terraform-014) | `TestAccAgynHook_basic` | @svc_gateway, @tf_provider_agyn |
| [E2E-GO-TERRAFORM-015](#e2e-go-terraform-015) | `TestAccAgynHook_update` | @svc_gateway, @tf_provider_agyn |
| [E2E-GO-TERRAFORM-016](#e2e-go-terraform-016) | `TestAccAgynHook_import` | @svc_gateway, @tf_provider_agyn |
| [E2E-GO-TERRAFORM-017](#e2e-go-terraform-017) | `TestAccAgynImagePullSecretAttachment_basic` | @svc_gateway, @tf_provider_agyn |
| [E2E-GO-TERRAFORM-018](#e2e-go-terraform-018) | `TestAccAgynImagePullSecretAttachment_update` | @svc_gateway, @tf_provider_agyn |
| [E2E-GO-TERRAFORM-019](#e2e-go-terraform-019) | `TestAccAgynImagePullSecretAttachment_import` | @svc_gateway, @tf_provider_agyn |
| [E2E-GO-TERRAFORM-020](#e2e-go-terraform-020) | `TestAccAgynImagePullSecret_basic` | @svc_gateway, @tf_provider_agyn |
| [E2E-GO-TERRAFORM-021](#e2e-go-terraform-021) | `TestAccAgynImagePullSecret_update` | @svc_gateway, @tf_provider_agyn |
| [E2E-GO-TERRAFORM-022](#e2e-go-terraform-022) | `TestAccAgynImagePullSecret_import` | @svc_gateway, @tf_provider_agyn |
| [E2E-GO-TERRAFORM-023](#e2e-go-terraform-023) | `TestAccAgynInitScript_basic` | @svc_gateway, @tf_provider_agyn |
| [E2E-GO-TERRAFORM-024](#e2e-go-terraform-024) | `TestAccAgynInitScript_update` | @svc_gateway, @tf_provider_agyn |
| [E2E-GO-TERRAFORM-025](#e2e-go-terraform-025) | `TestAccAgynInitScript_import` | @svc_gateway, @tf_provider_agyn |
| [E2E-GO-TERRAFORM-026](#e2e-go-terraform-026) | `TestAccAgynMcp_basic` | @svc_gateway, @tf_provider_agyn |
| [E2E-GO-TERRAFORM-027](#e2e-go-terraform-027) | `TestAccAgynMcp_update` | @svc_gateway, @tf_provider_agyn |
| [E2E-GO-TERRAFORM-028](#e2e-go-terraform-028) | `TestAccAgynMcp_import` | @svc_gateway, @tf_provider_agyn |
| [E2E-GO-TERRAFORM-029](#e2e-go-terraform-029) | `TestAccAgynOrganization_basic` | @svc_gateway, @tf_provider_agyn |
| [E2E-GO-TERRAFORM-030](#e2e-go-terraform-030) | `TestAccAgynOrganization_update` | @svc_gateway, @tf_provider_agyn |
| [E2E-GO-TERRAFORM-031](#e2e-go-terraform-031) | `TestAccAgynOrganization_import` | @svc_gateway, @tf_provider_agyn |
| [E2E-GO-TERRAFORM-032](#e2e-go-terraform-032) | `TestAccAgynRunner_basic` | @svc_gateway, @tf_provider_agyn |
| [E2E-GO-TERRAFORM-033](#e2e-go-terraform-033) | `TestAccAgynRunner_update` | @svc_gateway, @tf_provider_agyn |
| [E2E-GO-TERRAFORM-034](#e2e-go-terraform-034) | `TestAccAgynRunner_organizationIDRequiresReplace` | @svc_gateway, @tf_provider_agyn |
| [E2E-GO-TERRAFORM-035](#e2e-go-terraform-035) | `TestAccAgynRunner_import` | @svc_gateway, @tf_provider_agyn |
| [E2E-GO-TERRAFORM-036](#e2e-go-terraform-036) | `TestAccAgynSkill_basic` | @svc_gateway, @tf_provider_agyn |
| [E2E-GO-TERRAFORM-037](#e2e-go-terraform-037) | `TestAccAgynSkill_update` | @svc_gateway, @tf_provider_agyn |
| [E2E-GO-TERRAFORM-038](#e2e-go-terraform-038) | `TestAccAgynSkill_import` | @svc_gateway, @tf_provider_agyn |
| [E2E-GO-TERRAFORM-039](#e2e-go-terraform-039) | `TestAccAgynUser_basic` | @svc_gateway, @tf_provider_agyn |
| [E2E-GO-TERRAFORM-040](#e2e-go-terraform-040) | `TestAccAgynUser_update` | @svc_gateway, @tf_provider_agyn |
| [E2E-GO-TERRAFORM-041](#e2e-go-terraform-041) | `TestAccAgynUser_import` | @svc_gateway, @tf_provider_agyn |
| [E2E-GO-TERRAFORM-042](#e2e-go-terraform-042) | `TestAccAgynVolumeAttachment_basic` | @svc_gateway, @tf_provider_agyn |
| [E2E-GO-TERRAFORM-043](#e2e-go-terraform-043) | `TestAccAgynVolumeAttachment_update` | @svc_gateway, @tf_provider_agyn |
| [E2E-GO-TERRAFORM-044](#e2e-go-terraform-044) | `TestAccAgynVolumeAttachment_import` | @svc_gateway, @tf_provider_agyn |
| [E2E-GO-TERRAFORM-045](#e2e-go-terraform-045) | `TestAccAgynVolume_basic` | @svc_gateway, @tf_provider_agyn |
| [E2E-GO-TERRAFORM-046](#e2e-go-terraform-046) | `TestAccAgynVolume_update` | @svc_gateway, @tf_provider_agyn |
| [E2E-GO-TERRAFORM-047](#e2e-go-terraform-047) | `TestAccAgynVolume_import` | @svc_gateway, @tf_provider_agyn |

## Scenarios

### E2E-GO-TERRAFORM-001

- **Source:** `suites/go-terraform/tests/resource_agent_test.go`
- **Test:** `TestAccAgynAgent_basic`
- **Tags:** @svc_gateway, @tf_provider_agyn

**Scenario:** TestAccAgynAgent_basic

- **Given** Terraform provider credentials and prerequisite Agyn resources for Agent are configured.
- **When** Terraform applies the Agent basic acceptance scenario.
- **Then** The provider creates an agent with the requested name, role, labels, and internal availability.

### E2E-GO-TERRAFORM-002

- **Source:** `suites/go-terraform/tests/resource_agent_test.go`
- **Test:** `TestAccAgynAgent_update`
- **Tags:** @svc_gateway, @tf_provider_agyn

**Scenario:** TestAccAgynAgent_update

- **Given** Terraform provider credentials and prerequisite Agyn resources for Agent are configured.
- **When** Terraform applies the Agent update acceptance scenario.
- **Then** The provider updates agent fields including name, description, role, nickname, labels, and availability.

### E2E-GO-TERRAFORM-003

- **Source:** `suites/go-terraform/tests/resource_agent_test.go`
- **Test:** `TestAccAgynAgent_import`
- **Tags:** @svc_gateway, @tf_provider_agyn

**Scenario:** TestAccAgynAgent_import

- **Given** Terraform provider credentials and prerequisite Agyn resources for Agent are configured.
- **When** Terraform applies the Agent import acceptance scenario.
- **Then** The provider imports an existing agent and verifies Terraform state.

### E2E-GO-TERRAFORM-004

- **Source:** `suites/go-terraform/tests/resource_agent_test.go`
- **Test:** `TestAccAgynAgent_expectError`
- **Tags:** @svc_gateway, @tf_provider_agyn

**Scenario:** TestAccAgynAgent_expectError

- **Given** Terraform provider credentials and prerequisite Agyn resources for Agent are configured.
- **When** Terraform applies the Agent expectError acceptance scenario.
- **Then** The provider rejects invalid JSON in agent configuration.

### E2E-GO-TERRAFORM-005

- **Source:** `suites/go-terraform/tests/resource_agent_test.go`
- **Test:** `TestAccAgynAgent_invalidNickname`
- **Tags:** @svc_gateway, @tf_provider_agyn

**Scenario:** TestAccAgynAgent_invalidNickname

- **Given** Terraform provider credentials and prerequisite Agyn resources for Agent are configured.
- **When** Terraform applies the Agent invalidNickname acceptance scenario.
- **Then** The provider rejects agent nicknames that do not meet lowercase validation rules.

### E2E-GO-TERRAFORM-006

- **Source:** `suites/go-terraform/tests/resource_app_installation_test.go`
- **Test:** `TestAccAgynAppInstallation_basic`
- **Tags:** @svc_gateway, @tf_provider_agyn

**Scenario:** TestAccAgynAppInstallation_basic

- **Given** Terraform provider credentials and prerequisite Agyn resources for AppInstallation are configured.
- **When** Terraform applies the AppInstallation basic acceptance scenario.
- **Then** The provider creates an app installation with the requested slug and configuration.

### E2E-GO-TERRAFORM-007

- **Source:** `suites/go-terraform/tests/resource_app_installation_test.go`
- **Test:** `TestAccAgynAppInstallation_update`
- **Tags:** @svc_gateway, @tf_provider_agyn

**Scenario:** TestAccAgynAppInstallation_update

- **Given** Terraform provider credentials and prerequisite Agyn resources for AppInstallation are configured.
- **When** Terraform applies the AppInstallation update acceptance scenario.
- **Then** The provider updates app installation slug and configuration.

### E2E-GO-TERRAFORM-008

- **Source:** `suites/go-terraform/tests/resource_app_installation_test.go`
- **Test:** `TestAccAgynAppInstallation_import`
- **Tags:** @svc_gateway, @tf_provider_agyn

**Scenario:** TestAccAgynAppInstallation_import

- **Given** Terraform provider credentials and prerequisite Agyn resources for AppInstallation are configured.
- **When** Terraform applies the AppInstallation import acceptance scenario.
- **Then** The provider imports an existing app installation and verifies Terraform state.

### E2E-GO-TERRAFORM-009

- **Source:** `suites/go-terraform/tests/resource_app_test.go`
- **Test:** `TestAccAgynApp_basic`
- **Tags:** @svc_gateway, @tf_provider_agyn

**Scenario:** TestAccAgynApp_basic

- **Given** Terraform provider credentials and prerequisite Agyn resources for App are configured.
- **When** Terraform applies the App basic acceptance scenario.
- **Then** The provider creates an app with metadata and service token.

### E2E-GO-TERRAFORM-010

- **Source:** `suites/go-terraform/tests/resource_app_test.go`
- **Test:** `TestAccAgynApp_import`
- **Tags:** @svc_gateway, @tf_provider_agyn

**Scenario:** TestAccAgynApp_import

- **Given** Terraform provider credentials and prerequisite Agyn resources for App are configured.
- **When** Terraform applies the App import acceptance scenario.
- **Then** The provider imports an existing app and verifies Terraform state while ignoring service token.

### E2E-GO-TERRAFORM-011

- **Source:** `suites/go-terraform/tests/resource_env_test.go`
- **Test:** `TestAccAgynEnv_basic`
- **Tags:** @svc_gateway, @tf_provider_agyn

**Scenario:** TestAccAgynEnv_basic

- **Given** Terraform provider credentials and prerequisite Agyn resources for Env are configured.
- **When** Terraform applies the Env basic acceptance scenario.
- **Then** The provider creates an agent environment variable.

### E2E-GO-TERRAFORM-012

- **Source:** `suites/go-terraform/tests/resource_env_test.go`
- **Test:** `TestAccAgynEnv_update`
- **Tags:** @svc_gateway, @tf_provider_agyn

**Scenario:** TestAccAgynEnv_update

- **Given** Terraform provider credentials and prerequisite Agyn resources for Env are configured.
- **When** Terraform applies the Env update acceptance scenario.
- **Then** The provider updates the environment variable value and description.

### E2E-GO-TERRAFORM-013

- **Source:** `suites/go-terraform/tests/resource_env_test.go`
- **Test:** `TestAccAgynEnv_import`
- **Tags:** @svc_gateway, @tf_provider_agyn

**Scenario:** TestAccAgynEnv_import

- **Given** Terraform provider credentials and prerequisite Agyn resources for Env are configured.
- **When** Terraform applies the Env import acceptance scenario.
- **Then** The provider imports an existing environment variable and verifies Terraform state.

### E2E-GO-TERRAFORM-014

- **Source:** `suites/go-terraform/tests/resource_hook_test.go`
- **Test:** `TestAccAgynHook_basic`
- **Tags:** @svc_gateway, @tf_provider_agyn

**Scenario:** TestAccAgynHook_basic

- **Given** Terraform provider credentials and prerequisite Agyn resources for Hook are configured.
- **When** Terraform applies the Hook basic acceptance scenario.
- **Then** The provider creates an agent hook.

### E2E-GO-TERRAFORM-015

- **Source:** `suites/go-terraform/tests/resource_hook_test.go`
- **Test:** `TestAccAgynHook_update`
- **Tags:** @svc_gateway, @tf_provider_agyn

**Scenario:** TestAccAgynHook_update

- **Given** Terraform provider credentials and prerequisite Agyn resources for Hook are configured.
- **When** Terraform applies the Hook update acceptance scenario.
- **Then** The provider updates hook fields.

### E2E-GO-TERRAFORM-016

- **Source:** `suites/go-terraform/tests/resource_hook_test.go`
- **Test:** `TestAccAgynHook_import`
- **Tags:** @svc_gateway, @tf_provider_agyn

**Scenario:** TestAccAgynHook_import

- **Given** Terraform provider credentials and prerequisite Agyn resources for Hook are configured.
- **When** Terraform applies the Hook import acceptance scenario.
- **Then** The provider imports an existing hook and verifies Terraform state.

### E2E-GO-TERRAFORM-017

- **Source:** `suites/go-terraform/tests/resource_image_pull_secret_attachment_test.go`
- **Test:** `TestAccAgynImagePullSecretAttachment_basic`
- **Tags:** @svc_gateway, @tf_provider_agyn

**Scenario:** TestAccAgynImagePullSecretAttachment_basic

- **Given** Terraform provider credentials and prerequisite Agyn resources for ImagePullSecretAttachment are configured.
- **When** Terraform applies the ImagePullSecretAttachment basic acceptance scenario.
- **Then** The provider attaches an image pull secret.

### E2E-GO-TERRAFORM-018

- **Source:** `suites/go-terraform/tests/resource_image_pull_secret_attachment_test.go`
- **Test:** `TestAccAgynImagePullSecretAttachment_update`
- **Tags:** @svc_gateway, @tf_provider_agyn

**Scenario:** TestAccAgynImagePullSecretAttachment_update

- **Given** Terraform provider credentials and prerequisite Agyn resources for ImagePullSecretAttachment are configured.
- **When** Terraform applies the ImagePullSecretAttachment update acceptance scenario.
- **Then** The provider moves or updates the image pull secret attachment.

### E2E-GO-TERRAFORM-019

- **Source:** `suites/go-terraform/tests/resource_image_pull_secret_attachment_test.go`
- **Test:** `TestAccAgynImagePullSecretAttachment_import`
- **Tags:** @svc_gateway, @tf_provider_agyn

**Scenario:** TestAccAgynImagePullSecretAttachment_import

- **Given** Terraform provider credentials and prerequisite Agyn resources for ImagePullSecretAttachment are configured.
- **When** Terraform applies the ImagePullSecretAttachment import acceptance scenario.
- **Then** The provider imports an existing image pull secret attachment and verifies Terraform state.

### E2E-GO-TERRAFORM-020

- **Source:** `suites/go-terraform/tests/resource_image_pull_secret_test.go`
- **Test:** `TestAccAgynImagePullSecret_basic`
- **Tags:** @svc_gateway, @tf_provider_agyn

**Scenario:** TestAccAgynImagePullSecret_basic

- **Given** Terraform provider credentials and prerequisite Agyn resources for ImagePullSecret are configured.
- **When** Terraform applies the ImagePullSecret basic acceptance scenario.
- **Then** The provider creates an image pull secret.

### E2E-GO-TERRAFORM-021

- **Source:** `suites/go-terraform/tests/resource_image_pull_secret_test.go`
- **Test:** `TestAccAgynImagePullSecret_update`
- **Tags:** @svc_gateway, @tf_provider_agyn

**Scenario:** TestAccAgynImagePullSecret_update

- **Given** Terraform provider credentials and prerequisite Agyn resources for ImagePullSecret are configured.
- **When** Terraform applies the ImagePullSecret update acceptance scenario.
- **Then** The provider updates image pull secret metadata and secret value.

### E2E-GO-TERRAFORM-022

- **Source:** `suites/go-terraform/tests/resource_image_pull_secret_test.go`
- **Test:** `TestAccAgynImagePullSecret_import`
- **Tags:** @svc_gateway, @tf_provider_agyn

**Scenario:** TestAccAgynImagePullSecret_import

- **Given** Terraform provider credentials and prerequisite Agyn resources for ImagePullSecret are configured.
- **When** Terraform applies the ImagePullSecret import acceptance scenario.
- **Then** The provider imports an existing image pull secret and verifies Terraform state.

### E2E-GO-TERRAFORM-023

- **Source:** `suites/go-terraform/tests/resource_init_script_test.go`
- **Test:** `TestAccAgynInitScript_basic`
- **Tags:** @svc_gateway, @tf_provider_agyn

**Scenario:** TestAccAgynInitScript_basic

- **Given** Terraform provider credentials and prerequisite Agyn resources for InitScript are configured.
- **When** Terraform applies the InitScript basic acceptance scenario.
- **Then** The provider creates an agent init script.

### E2E-GO-TERRAFORM-024

- **Source:** `suites/go-terraform/tests/resource_init_script_test.go`
- **Test:** `TestAccAgynInitScript_update`
- **Tags:** @svc_gateway, @tf_provider_agyn

**Scenario:** TestAccAgynInitScript_update

- **Given** Terraform provider credentials and prerequisite Agyn resources for InitScript are configured.
- **When** Terraform applies the InitScript update acceptance scenario.
- **Then** The provider updates init script content or metadata.

### E2E-GO-TERRAFORM-025

- **Source:** `suites/go-terraform/tests/resource_init_script_test.go`
- **Test:** `TestAccAgynInitScript_import`
- **Tags:** @svc_gateway, @tf_provider_agyn

**Scenario:** TestAccAgynInitScript_import

- **Given** Terraform provider credentials and prerequisite Agyn resources for InitScript are configured.
- **When** Terraform applies the InitScript import acceptance scenario.
- **Then** The provider imports an existing init script and verifies Terraform state.

### E2E-GO-TERRAFORM-026

- **Source:** `suites/go-terraform/tests/resource_mcp_test.go`
- **Test:** `TestAccAgynMcp_basic`
- **Tags:** @svc_gateway, @tf_provider_agyn

**Scenario:** TestAccAgynMcp_basic

- **Given** Terraform provider credentials and prerequisite Agyn resources for Mcp are configured.
- **When** Terraform applies the Mcp basic acceptance scenario.
- **Then** The provider creates an agent MCP server definition.

### E2E-GO-TERRAFORM-027

- **Source:** `suites/go-terraform/tests/resource_mcp_test.go`
- **Test:** `TestAccAgynMcp_update`
- **Tags:** @svc_gateway, @tf_provider_agyn

**Scenario:** TestAccAgynMcp_update

- **Given** Terraform provider credentials and prerequisite Agyn resources for Mcp are configured.
- **When** Terraform applies the Mcp update acceptance scenario.
- **Then** The provider updates the MCP definition.

### E2E-GO-TERRAFORM-028

- **Source:** `suites/go-terraform/tests/resource_mcp_test.go`
- **Test:** `TestAccAgynMcp_import`
- **Tags:** @svc_gateway, @tf_provider_agyn

**Scenario:** TestAccAgynMcp_import

- **Given** Terraform provider credentials and prerequisite Agyn resources for Mcp are configured.
- **When** Terraform applies the Mcp import acceptance scenario.
- **Then** The provider imports an existing MCP definition and verifies Terraform state.

### E2E-GO-TERRAFORM-029

- **Source:** `suites/go-terraform/tests/resource_organization_test.go`
- **Test:** `TestAccAgynOrganization_basic`
- **Tags:** @svc_gateway, @tf_provider_agyn

**Scenario:** TestAccAgynOrganization_basic

- **Given** Terraform provider credentials and prerequisite Agyn resources for Organization are configured.
- **When** Terraform applies the Organization basic acceptance scenario.
- **Then** The provider creates an organization.

### E2E-GO-TERRAFORM-030

- **Source:** `suites/go-terraform/tests/resource_organization_test.go`
- **Test:** `TestAccAgynOrganization_update`
- **Tags:** @svc_gateway, @tf_provider_agyn

**Scenario:** TestAccAgynOrganization_update

- **Given** Terraform provider credentials and prerequisite Agyn resources for Organization are configured.
- **When** Terraform applies the Organization update acceptance scenario.
- **Then** The provider updates organization metadata.

### E2E-GO-TERRAFORM-031

- **Source:** `suites/go-terraform/tests/resource_organization_test.go`
- **Test:** `TestAccAgynOrganization_import`
- **Tags:** @svc_gateway, @tf_provider_agyn

**Scenario:** TestAccAgynOrganization_import

- **Given** Terraform provider credentials and prerequisite Agyn resources for Organization are configured.
- **When** Terraform applies the Organization import acceptance scenario.
- **Then** The provider imports an existing organization and verifies Terraform state.

### E2E-GO-TERRAFORM-032

- **Source:** `suites/go-terraform/tests/resource_runner_test.go`
- **Test:** `TestAccAgynRunner_basic`
- **Tags:** @svc_gateway, @tf_provider_agyn

**Scenario:** TestAccAgynRunner_basic

- **Given** Terraform provider credentials and prerequisite Agyn resources for Runner are configured.
- **When** Terraform applies the Runner basic acceptance scenario.
- **Then** The provider creates a runner.

### E2E-GO-TERRAFORM-033

- **Source:** `suites/go-terraform/tests/resource_runner_test.go`
- **Test:** `TestAccAgynRunner_update`
- **Tags:** @svc_gateway, @tf_provider_agyn

**Scenario:** TestAccAgynRunner_update

- **Given** Terraform provider credentials and prerequisite Agyn resources for Runner are configured.
- **When** Terraform applies the Runner update acceptance scenario.
- **Then** The provider updates runner labels and metadata.

### E2E-GO-TERRAFORM-034

- **Source:** `suites/go-terraform/tests/resource_runner_test.go`
- **Test:** `TestAccAgynRunner_organizationIDRequiresReplace`
- **Tags:** @svc_gateway, @tf_provider_agyn

**Scenario:** TestAccAgynRunner_organizationIDRequiresReplace

- **Given** Terraform provider credentials and prerequisite Agyn resources for Runner are configured.
- **When** Terraform applies the Runner organizationIDRequiresReplace acceptance scenario.
- **Then** The provider replaces the runner when organization ownership changes.

### E2E-GO-TERRAFORM-035

- **Source:** `suites/go-terraform/tests/resource_runner_test.go`
- **Test:** `TestAccAgynRunner_import`
- **Tags:** @svc_gateway, @tf_provider_agyn

**Scenario:** TestAccAgynRunner_import

- **Given** Terraform provider credentials and prerequisite Agyn resources for Runner are configured.
- **When** Terraform applies the Runner import acceptance scenario.
- **Then** The provider imports an existing runner and verifies Terraform state while ignoring service token.

### E2E-GO-TERRAFORM-036

- **Source:** `suites/go-terraform/tests/resource_skill_test.go`
- **Test:** `TestAccAgynSkill_basic`
- **Tags:** @svc_gateway, @tf_provider_agyn

**Scenario:** TestAccAgynSkill_basic

- **Given** Terraform provider credentials and prerequisite Agyn resources for Skill are configured.
- **When** Terraform applies the Skill basic acceptance scenario.
- **Then** The provider creates an agent skill.

### E2E-GO-TERRAFORM-037

- **Source:** `suites/go-terraform/tests/resource_skill_test.go`
- **Test:** `TestAccAgynSkill_update`
- **Tags:** @svc_gateway, @tf_provider_agyn

**Scenario:** TestAccAgynSkill_update

- **Given** Terraform provider credentials and prerequisite Agyn resources for Skill are configured.
- **When** Terraform applies the Skill update acceptance scenario.
- **Then** The provider updates the skill name, description, or command.

### E2E-GO-TERRAFORM-038

- **Source:** `suites/go-terraform/tests/resource_skill_test.go`
- **Test:** `TestAccAgynSkill_import`
- **Tags:** @svc_gateway, @tf_provider_agyn

**Scenario:** TestAccAgynSkill_import

- **Given** Terraform provider credentials and prerequisite Agyn resources for Skill are configured.
- **When** Terraform applies the Skill import acceptance scenario.
- **Then** The provider imports an existing skill and verifies Terraform state.

### E2E-GO-TERRAFORM-039

- **Source:** `suites/go-terraform/tests/resource_user_test.go`
- **Test:** `TestAccAgynUser_basic`
- **Tags:** @svc_gateway, @tf_provider_agyn

**Scenario:** TestAccAgynUser_basic

- **Given** Terraform provider credentials and prerequisite Agyn resources for User are configured.
- **When** Terraform applies the User basic acceptance scenario.
- **Then** The provider creates a user with the requested role.

### E2E-GO-TERRAFORM-040

- **Source:** `suites/go-terraform/tests/resource_user_test.go`
- **Test:** `TestAccAgynUser_update`
- **Tags:** @svc_gateway, @tf_provider_agyn

**Scenario:** TestAccAgynUser_update

- **Given** Terraform provider credentials and prerequisite Agyn resources for User are configured.
- **When** Terraform applies the User update acceptance scenario.
- **Then** The provider updates user profile or role fields.

### E2E-GO-TERRAFORM-041

- **Source:** `suites/go-terraform/tests/resource_user_test.go`
- **Test:** `TestAccAgynUser_import`
- **Tags:** @svc_gateway, @tf_provider_agyn

**Scenario:** TestAccAgynUser_import

- **Given** Terraform provider credentials and prerequisite Agyn resources for User are configured.
- **When** Terraform applies the User import acceptance scenario.
- **Then** The provider imports an existing user and verifies Terraform state by identity id.

### E2E-GO-TERRAFORM-042

- **Source:** `suites/go-terraform/tests/resource_volume_attachment_test.go`
- **Test:** `TestAccAgynVolumeAttachment_basic`
- **Tags:** @svc_gateway, @tf_provider_agyn

**Scenario:** TestAccAgynVolumeAttachment_basic

- **Given** Terraform provider credentials and prerequisite Agyn resources for VolumeAttachment are configured.
- **When** Terraform applies the VolumeAttachment basic acceptance scenario.
- **Then** The provider attaches a volume to an agent.

### E2E-GO-TERRAFORM-043

- **Source:** `suites/go-terraform/tests/resource_volume_attachment_test.go`
- **Test:** `TestAccAgynVolumeAttachment_update`
- **Tags:** @svc_gateway, @tf_provider_agyn

**Scenario:** TestAccAgynVolumeAttachment_update

- **Given** Terraform provider credentials and prerequisite Agyn resources for VolumeAttachment are configured.
- **When** Terraform applies the VolumeAttachment update acceptance scenario.
- **Then** The provider updates the attachment read-only setting.

### E2E-GO-TERRAFORM-044

- **Source:** `suites/go-terraform/tests/resource_volume_attachment_test.go`
- **Test:** `TestAccAgynVolumeAttachment_import`
- **Tags:** @svc_gateway, @tf_provider_agyn

**Scenario:** TestAccAgynVolumeAttachment_import

- **Given** Terraform provider credentials and prerequisite Agyn resources for VolumeAttachment are configured.
- **When** Terraform applies the VolumeAttachment import acceptance scenario.
- **Then** The provider imports an existing volume attachment and verifies Terraform state.

### E2E-GO-TERRAFORM-045

- **Source:** `suites/go-terraform/tests/resource_volume_test.go`
- **Test:** `TestAccAgynVolume_basic`
- **Tags:** @svc_gateway, @tf_provider_agyn

**Scenario:** TestAccAgynVolume_basic

- **Given** Terraform provider credentials and prerequisite Agyn resources for Volume are configured.
- **When** Terraform applies the Volume basic acceptance scenario.
- **Then** The provider creates a volume.

### E2E-GO-TERRAFORM-046

- **Source:** `suites/go-terraform/tests/resource_volume_test.go`
- **Test:** `TestAccAgynVolume_update`
- **Tags:** @svc_gateway, @tf_provider_agyn

**Scenario:** TestAccAgynVolume_update

- **Given** Terraform provider credentials and prerequisite Agyn resources for Volume are configured.
- **When** Terraform applies the Volume update acceptance scenario.
- **Then** The provider updates volume description, mount path, and size.

### E2E-GO-TERRAFORM-047

- **Source:** `suites/go-terraform/tests/resource_volume_test.go`
- **Test:** `TestAccAgynVolume_import`
- **Tags:** @svc_gateway, @tf_provider_agyn

**Scenario:** TestAccAgynVolume_import

- **Given** Terraform provider credentials and prerequisite Agyn resources for Volume are configured.
- **When** Terraform applies the Volume import acceptance scenario.
- **Then** The provider imports an existing volume and verifies Terraform state while ignoring organization id.
