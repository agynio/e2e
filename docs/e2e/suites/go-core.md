# Go Core

- **Suite ID:** E2E-SUITE-GO-CORE
- **Source:** `suites/go-core/suite.yaml`
- **Kind:** Go
- **Tags:** @svc_agents_orchestrator, @svc_runners, @svc_metering, @svc_k8s_runner, @svc_organizations, @svc_files, @svc_gateway, @svc_egress, @svc_egress_gateway, @svc_media_proxy, @svc_llm, @svc_llm_proxy, @smoke

## Intent

Validates core platform services through Go E2E tests: gateway authentication, agents, organizations, runners, workloads, files, egress control-plane, gateway wiring, and data-plane matching, media proxying, LLM proxying, metering, tracing, MCP tools, Ziti exposure, and retry behavior.

## Scope

- Source directory: `suites/go-core`
- Test inventory pattern: `tests/*_test.go`
- Included case count: 94

## Actors

- Platform API client
- Authenticated user
- Agent runtime
- Kubernetes runner
- Gateway
- Files service
- Media proxy
- LLM proxy
- Metering service
- Tracing service

## Preconditions

- A provisioned Agyn platform and Kubernetes runner are available.
- Required service endpoints, credentials, images, and init images are configured.
- Optional tracing-dependent cases run only when tracing is available.

## Tags

- `@svc_agents_orchestrator`
- `@svc_runners`
- `@svc_metering`
- `@svc_k8s_runner`
- `@svc_organizations`
- `@svc_files`
- `@svc_gateway`
- `@svc_egress`
- `@svc_egress_gateway`
- `@svc_media_proxy`
- `@svc_llm`
- `@svc_llm_proxy`
- `@smoke`

## Case index

| Case ID | Source test | Tags |
| --- | --- | --- |
| [E2E-GO-CORE-001](#e2e-go-core-001) | `TestAgentAgynCLIWaitToAnotherAgent` | @svc_agents_orchestrator |
| [E2E-GO-CORE-002](#e2e-go-core-002) | `TestNoDuplicateWorkloads` | @svc_agents_orchestrator, @svc_runners |
| [E2E-GO-CORE-003](#e2e-go-core-003) | `TestZitiManagementEndpointDefaultUsesIngress` | @svc_agents_orchestrator |
| [E2E-GO-CORE-004](#e2e-go-core-004) | `TestZitiManagementEndpointExplicitOverride` | @svc_agents_orchestrator |
| [E2E-GO-CORE-005](#e2e-go-core-005) | `TestAgentExposeListExec` | @svc_agents_orchestrator |
| [E2E-GO-CORE-006](#e2e-go-core-006) | `TestAgentExposeLifecycle_ListAddRemove` | @svc_agents_orchestrator, @svc_gateway |
| [E2E-GO-CORE-007](#e2e-go-core-007) | `TestFilesSmokeMetadataRequiresID` | @svc_files, @smoke |
| [E2E-GO-CORE-008](#e2e-go-core-008) | `TestFilesGetFileMetadataRequiresID` | @svc_files |
| [E2E-GO-CORE-009](#e2e-go-core-009) | `TestFilesGetFileContentRoundTrip` | @svc_files |
| [E2E-GO-CORE-010](#e2e-go-core-010) | `TestFilesGetFileContentNotFound` | @svc_files |
| [E2E-GO-CORE-011](#e2e-go-core-011) | `TestAgentsGateway_ListAgents` | @svc_gateway, @svc_agents_orchestrator |
| [E2E-GO-CORE-012](#e2e-go-core-012) | `TestAgentsGateway_CreateAndDeleteAgent` | @svc_gateway, @svc_agents_orchestrator |
| [E2E-GO-CORE-013](#e2e-go-core-013) | `TestAgentsGateway_ListMcps` | @svc_gateway, @svc_agents_orchestrator |
| [E2E-GO-CORE-014](#e2e-go-core-014) | `TestAgentsGateway_InvalidPayloadReturnsClientError` | @svc_gateway |
| [E2E-GO-CORE-015](#e2e-go-core-015) | `TestGatewayMeEndpointUnauthenticated` | @svc_gateway, @smoke |
| [E2E-GO-CORE-016](#e2e-go-core-016) | `TestGatewayMeEndpointAuthenticated` | @svc_gateway, @smoke |
| [E2E-GO-CORE-017](#e2e-go-core-017) | `TestAPIToken_MeEndpoint` | @svc_gateway |
| [E2E-GO-CORE-018](#e2e-go-core-018) | `TestAPIToken_MeEndpointInvalidToken` | @svc_gateway |
| [E2E-GO-CORE-019](#e2e-go-core-019) | `TestAPIToken_ConnectRPCEndpointAuthenticated` | @svc_gateway |
| [E2E-GO-CORE-020](#e2e-go-core-020) | `TestAPIToken_ConnectRPCEndpointInvalidToken` | @svc_gateway |
| [E2E-GO-CORE-021](#e2e-go-core-021) | `TestUsersGateway_CreateAndRevokeAPIToken` | @svc_gateway, @svc_identity |
| [E2E-GO-CORE-022](#e2e-go-core-022) | `TestUsersGateway_ListAPITokens` | @svc_gateway, @svc_identity |
| [E2E-GO-CORE-023](#e2e-go-core-023) | `TestUsersGateway_RevokeAPITokenNotFound` | @svc_gateway, @svc_identity |
| [E2E-GO-CORE-024](#e2e-go-core-024) | `TestUsersGateway_CreateAPITokenUnauthenticated` | @svc_gateway, @svc_identity |
| [E2E-GO-CORE-025](#e2e-go-core-025) | `TestAPIToken_CreatedTokenAuthenticates` | @svc_gateway, @svc_identity |
| [E2E-GO-CORE-026](#e2e-go-core-026) | `TestZitiMeEndpointAuthenticated` | @svc_gateway |
| [E2E-GO-CORE-027](#e2e-go-core-027) | `TestWorkloadStopsAfterIdleTimeout` | @svc_runners, @svc_k8s_runner |
| [E2E-GO-CORE-028](#e2e-go-core-028) | `TestImagePullSecretAttachedToPod` | @svc_runners, @svc_k8s_runner |
| [E2E-GO-CORE-029](#e2e-go-core-029) | `TestErrors/start_workload_missing_image` | @svc_k8s_runner |
| [E2E-GO-CORE-030](#e2e-go-core-030) | `TestErrors/inspect_nonexistent_workload` | @svc_k8s_runner |
| [E2E-GO-CORE-031](#e2e-go-core-031) | `TestErrors/stream_logs_missing_container_name` | @svc_k8s_runner |
| [E2E-GO-CORE-032](#e2e-go-core-032) | `TestErrors/exec_on_nonexistent_workload` | @svc_k8s_runner |
| [E2E-GO-CORE-033](#e2e-go-core-033) | `TestErrors/remove_nonexistent_volume` | @svc_k8s_runner |
| [E2E-GO-CORE-034](#e2e-go-core-034) | `TestExec/basic_command` | @svc_k8s_runner |
| [E2E-GO-CORE-035](#e2e-go-core-035) | `TestExec/shell_command` | @svc_k8s_runner |
| [E2E-GO-CORE-036](#e2e-go-core-036) | `TestExec/nonzero_exit_code` | @svc_k8s_runner |
| [E2E-GO-CORE-037](#e2e-go-core-037) | `TestExec/stdin_and_eof` | @svc_k8s_runner |
| [E2E-GO-CORE-038](#e2e-go-core-038) | `TestExec/cancel_execution` | @svc_k8s_runner |
| [E2E-GO-CORE-039](#e2e-go-core-039) | `TestExec/workdir_and_env` | @svc_k8s_runner |
| [E2E-GO-CORE-040](#e2e-go-core-040) | `TestPutArchive` | @svc_k8s_runner |
| [E2E-GO-CORE-041](#e2e-go-core-041) | `TestStreaming/logs_follow` | @svc_k8s_runner |
| [E2E-GO-CORE-042](#e2e-go-core-042) | `TestStreaming/logs_tail` | @svc_k8s_runner |
| [E2E-GO-CORE-043](#e2e-go-core-043) | `TestStreamEvents` | @svc_k8s_runner |
| [E2E-GO-CORE-044](#e2e-go-core-044) | `TestVolumeQueries/list_workloads_by_volume` | @svc_k8s_runner |
| [E2E-GO-CORE-045](#e2e-go-core-045) | `TestVolumeQueries/remove_volume` | @svc_k8s_runner |
| [E2E-GO-CORE-046](#e2e-go-core-046) | `TestReady` | @svc_k8s_runner, @smoke |
| [E2E-GO-CORE-047](#e2e-go-core-047) | `TestWorkloadLifecycle/start_and_inspect` | @svc_k8s_runner |
| [E2E-GO-CORE-048](#e2e-go-core-048) | `TestWorkloadLifecycle/start_with_env_and_workdir` | @svc_k8s_runner |
| [E2E-GO-CORE-049](#e2e-go-core-049) | `TestWorkloadLifecycle/start_with_sidecars` | @svc_k8s_runner |
| [E2E-GO-CORE-050](#e2e-go-core-050) | `TestWorkloadLifecycle/start_with_custom_labels` | @svc_k8s_runner |
| [E2E-GO-CORE-051](#e2e-go-core-051) | `TestWorkloadLifecycle/touch_workload` | @svc_k8s_runner |
| [E2E-GO-CORE-052](#e2e-go-core-052) | `TestWorkloadLifecycle/stop_workload` | @svc_k8s_runner |
| [E2E-GO-CORE-053](#e2e-go-core-053) | `TestWorkloadLifecycle/remove_workload_with_volumes` | @svc_k8s_runner |
| [E2E-GO-CORE-054](#e2e-go-core-054) | `TestLLMProxyURLDefaultUsesIngress` | @svc_llm_proxy |
| [E2E-GO-CORE-055](#e2e-go-core-055) | `TestLLMProxyURLExplicitOverride` | @svc_llm_proxy |
| [E2E-GO-CORE-056](#e2e-go-core-056) | `TestLLMGatewayConnectEndpointPath` | @svc_llm, @svc_llm_proxy |
| [E2E-GO-CORE-057](#e2e-go-core-057) | `TestLLMProxyGatewayCreatedModel` | @svc_llm, @svc_llm_proxy |
| [E2E-GO-CORE-058](#e2e-go-core-058) | `TestMCPToolsE2E` | @svc_agents_orchestrator |
| [E2E-GO-CORE-059](#e2e-go-core-059) | `TestMCPToolsAgnE2E` | @svc_agents_orchestrator |
| [E2E-GO-CORE-060](#e2e-go-core-060) | `TestExternalProxy_PublicImage` | @svc_media_proxy |
| [E2E-GO-CORE-061](#e2e-go-core-061) | `TestExternalProxy_PublicImageWithResize` | @svc_media_proxy |
| [E2E-GO-CORE-062](#e2e-go-core-062) | `TestExternalProxy_NonImageURL` | @svc_media_proxy |
| [E2E-GO-CORE-063](#e2e-go-core-063) | `TestExternalProxy_Unauthenticated` | @svc_media_proxy |
| [E2E-GO-CORE-064](#e2e-go-core-064) | `TestExternalProxy_InvalidToken` | @svc_media_proxy |
| [E2E-GO-CORE-065](#e2e-go-core-065) | `TestExternalProxy_MissingURL` | @svc_media_proxy |
| [E2E-GO-CORE-066](#e2e-go-core-066) | `TestFileProxy_UploadAndProxy` | @svc_media_proxy, @svc_files |
| [E2E-GO-CORE-067](#e2e-go-core-067) | `TestFileProxy_UploadAndProxyWithResize` | @svc_media_proxy, @svc_files |
| [E2E-GO-CORE-068](#e2e-go-core-068) | `TestFileProxy_NotFoundFile` | @svc_media_proxy, @svc_files |
| [E2E-GO-CORE-069](#e2e-go-core-069) | `TestFileProxy_RangeRequest` | @svc_media_proxy, @svc_files |
| [E2E-GO-CORE-070](#e2e-go-core-070) | `TestFileProxy_Unauthenticated` | @svc_media_proxy, @svc_files |
| [E2E-GO-CORE-071](#e2e-go-core-071) | `TestRecordAndQueryUsage` | @svc_metering |
| [E2E-GO-CORE-072](#e2e-go-core-072) | `TestMultipleAgentsSeparateThreads` | @svc_agents_orchestrator |
| [E2E-GO-CORE-073](#e2e-go-core-073) | `TestSameAgentMultipleThreads` | @svc_agents_orchestrator |
| [E2E-GO-CORE-074](#e2e-go-core-074) | `TestOrganizationsServiceE2E` | @svc_organizations |
| [E2E-GO-CORE-075](#e2e-go-core-075) | `TestFullPipelineMessageResponse` | @svc_agents_orchestrator, @svc_llm, @smoke |
| [E2E-GO-CORE-082](#e2e-go-core-082) | `TestAgentSimpleHelloProducesTrace` | @svc_agents_orchestrator |
| [E2E-GO-CORE-083](#e2e-go-core-083) | `TestAgentMCPToolsProducesTrace` | @svc_agents_orchestrator |
| [E2E-GO-CORE-084](#e2e-go-core-084) | `TestWorkloadStartRetryPolicyFastRetry` | @svc_agents_orchestrator, @svc_runners, @svc_k8s_runner |
| [E2E-GO-CORE-087](#e2e-go-core-087) | `TestFullPipelineAgnMessageResponse` | @svc_agents_orchestrator, @svc_llm |
| [E2E-GO-CORE-088](#e2e-go-core-088) | `TestFullPipelineClaudeMessageResponse` | @svc_agents_orchestrator, @svc_llm |
| [E2E-GO-CORE-089](#e2e-go-core-089) | `TestBatchUpdateWorkloadSampledAtSingle` | @svc_runners |
| [E2E-GO-CORE-090](#e2e-go-core-090) | `TestBatchUpdateWorkloadSampledAtMultiple` | @svc_runners |
| [E2E-GO-CORE-091](#e2e-go-core-091) | `TestBatchUpdateWorkloadSampledAtIdempotent` | @svc_runners |
| [E2E-GO-CORE-092](#e2e-go-core-092) | `TestBatchUpdateVolumeSampledAtSingle` | @svc_runners |
| [E2E-GO-CORE-093](#e2e-go-core-093) | `TestBatchUpdateVolumeSampledAtMultiple` | @svc_runners |
| [E2E-GO-CORE-094](#e2e-go-core-094) | `TestBatchUpdateVolumeSampledAtIdempotent` | @svc_runners |
| [E2E-GO-CORE-095](#e2e-go-core-095) | `TestRunnerLifecycle` | @svc_runners |
| [E2E-GO-CORE-096](#e2e-go-core-096) | `TestWorkloadStartsOnUnackedMessage` | @svc_agents_orchestrator, @svc_runners |
| [E2E-GO-CORE-097](#e2e-go-core-097) | `TestThreadsSendShell` | @svc_agents_orchestrator |
| [E2E-GO-CORE-098](#e2e-go-core-098) | `TestEgressGatewayFeaturePath` | @svc_egress |
| [E2E-GO-CORE-099](#e2e-go-core-099) | `TestEgressGatewayDenyAndNoRulePaths` | @svc_egress |
| [E2E-GO-CORE-100](#e2e-go-core-100) | `TestEgressGatewayDeploymentWiring` | @svc_egress_gateway |
| [E2E-GO-CORE-101](#e2e-go-core-101) | `TestEgressGatewayDataPlaneHTTPBehavior` | @svc_egress_gateway |
| [E2E-GO-CORE-102](#e2e-go-core-102) | `TestEgressGatewayDataPlaneMatcherMatrix` | @svc_egress_gateway |

## Scenarios

### E2E-GO-CORE-001

- **Source:** `suites/go-core/tests/agent_agyn_wait_test.go`
- **Test:** `TestAgentAgynCLIWaitToAnotherAgent`
- **Tags:** @svc_agents_orchestrator

**Scenario:** TestAgentAgynCLIWaitToAnotherAgent

- **Given** Two agents can communicate through threads.
- **When** One agent waits for another agent to respond in a shared thread.
- **Then** The expected message is found in the active thread between the participants.
- **And** A real workload pod exists and its Ziti enrollment, gateway wait, and restartable sidecar containers are healthy.

### E2E-GO-CORE-002

- **Source:** `suites/go-core/tests/dedup_test.go`
- **Test:** `TestNoDuplicateWorkloads`
- **Tags:** @svc_agents_orchestrator, @svc_runners

**Scenario:** TestNoDuplicateWorkloads

- **Given** An agent request is submitted to the orchestrator.
- **When** Workload creation is retried or observed during processing.
- **Then** At most one workload exists for the same unit of work.

### E2E-GO-CORE-003

- **Source:** `suites/go-core/tests/expose_test.go`
- **Test:** `TestZitiManagementEndpointDefaultUsesIngress`
- **Tags:** @svc_agents_orchestrator

**Scenario:** TestZitiManagementEndpointDefaultUsesIngress

- **Given** Ingress domain and port environment values are set.
- **When** The Ziti management endpoint is built without an explicit override.
- **Then** The endpoint uses the ingress host and management API path.

### E2E-GO-CORE-004

- **Source:** `suites/go-core/tests/expose_test.go`
- **Test:** `TestZitiManagementEndpointExplicitOverride`
- **Tags:** @svc_agents_orchestrator

**Scenario:** TestZitiManagementEndpointExplicitOverride

- **Given** An explicit Ziti management endpoint override is set.
- **When** The endpoint builder is called.
- **Then** The explicit override is used unchanged with the management API path.

### E2E-GO-CORE-005

- **Source:** `suites/go-core/tests/expose_test.go`
- **Test:** `TestAgentExposeListExec`
- **Tags:** @svc_agents_orchestrator

**Scenario:** TestAgentExposeListExec

- **Given** An agent exposes commands through the AGN CLI.
- **When** A user lists exposed services through agent execution.
- **Then** The expose list command completes successfully and returns parseable output. TODO: clarify.

### E2E-GO-CORE-006

- **Source:** `suites/go-core/tests/expose_test.go`
- **Test:** `TestAgentExposeLifecycle_ListAddRemove`
- **Tags:** @svc_agents_orchestrator, @svc_gateway

**Scenario:** TestAgentExposeLifecycle_ListAddRemove

- **Given** An agent can expose a reachable HTTP service through Ziti.
- **When** A user lists exposes, adds an expose, verifies reachability, and removes it.
- **Then** The expose appears with the expected URL/status, serves the expected response, and is absent after removal.

### E2E-GO-CORE-007

- **Source:** `suites/go-core/tests/files_smoke_test.go`
- **Test:** `TestFilesSmokeMetadataRequiresID`
- **Tags:** @svc_files, @smoke

**Scenario:** TestFilesSmokeMetadataRequiresID

- **Given** A caller invokes the files metadata API without an id.
- **When** The request is sent through the smoke path.
- **Then** The service rejects the request with a clear client error.

### E2E-GO-CORE-008

- **Source:** `suites/go-core/tests/files_test.go`
- **Test:** `TestFilesGetFileMetadataRequiresID`
- **Tags:** @svc_files

**Scenario:** TestFilesGetFileMetadataRequiresID

- **Given** A caller invokes `GetFileMetadata` without a file id.
- **When** The request reaches the files service.
- **Then** The service returns a validation error instead of metadata.

### E2E-GO-CORE-009

- **Source:** `suites/go-core/tests/files_test.go`
- **Test:** `TestFilesGetFileContentRoundTrip`
- **Tags:** @svc_files

**Scenario:** TestFilesGetFileContentRoundTrip

- **Given** A file is uploaded to the files service.
- **When** A caller retrieves the file content by id.
- **Then** The downloaded content matches the uploaded content.

### E2E-GO-CORE-010

- **Source:** `suites/go-core/tests/files_test.go`
- **Test:** `TestFilesGetFileContentNotFound`
- **Tags:** @svc_files

**Scenario:** TestFilesGetFileContentNotFound

- **Given** No file exists for a requested id.
- **When** A caller requests file content for that id.
- **Then** The files service returns a not-found response.

### E2E-GO-CORE-011

- **Source:** `suites/go-core/tests/gateway_agents_test.go`
- **Test:** `TestAgentsGateway_ListAgents`
- **Tags:** @svc_gateway, @svc_agents_orchestrator

**Scenario:** TestAgentsGateway_ListAgents

- **Given** The gateway has access to agents for an organization.
- **When** A client lists agents through the gateway.
- **Then** The gateway returns the organization agents.

### E2E-GO-CORE-012

- **Source:** `suites/go-core/tests/gateway_agents_test.go`
- **Test:** `TestAgentsGateway_CreateAndDeleteAgent`
- **Tags:** @svc_gateway, @svc_agents_orchestrator

**Scenario:** TestAgentsGateway_CreateAndDeleteAgent

- **Given** A client can manage agents through the gateway.
- **When** The client creates an agent and then deletes it.
- **Then** The created agent is returned and is no longer listed after deletion.

### E2E-GO-CORE-013

- **Source:** `suites/go-core/tests/gateway_agents_test.go`
- **Test:** `TestAgentsGateway_ListMcps`
- **Tags:** @svc_gateway, @svc_agents_orchestrator

**Scenario:** TestAgentsGateway_ListMcps

- **Given** An agent has MCP definitions.
- **When** A client lists MCPs through the gateway.
- **Then** The gateway returns the expected MCP records.

### E2E-GO-CORE-014

- **Source:** `suites/go-core/tests/gateway_agents_test.go`
- **Test:** `TestAgentsGateway_InvalidPayloadReturnsClientError`
- **Tags:** @svc_gateway

**Scenario:** TestAgentsGateway_InvalidPayloadReturnsClientError

- **Given** A client sends an invalid gateway payload.
- **When** The gateway handles the request.
- **Then** The gateway returns a client error rather than accepting the payload.

### E2E-GO-CORE-015

- **Source:** `suites/go-core/tests/gateway_smoke_test.go`
- **Test:** `TestGatewayMeEndpointUnauthenticated`
- **Tags:** @svc_gateway, @smoke

**Scenario:** TestGatewayMeEndpointUnauthenticated

- **Given** No credentials are supplied.
- **When** A caller requests the gateway `/me` endpoint.
- **Then** The gateway rejects the request as unauthenticated.

### E2E-GO-CORE-016

- **Source:** `suites/go-core/tests/gateway_smoke_test.go`
- **Test:** `TestGatewayMeEndpointAuthenticated`
- **Tags:** @svc_gateway, @smoke

**Scenario:** TestGatewayMeEndpointAuthenticated

- **Given** A valid user credential is supplied.
- **When** The caller requests the gateway `/me` endpoint.
- **Then** The gateway returns the authenticated user context.

### E2E-GO-CORE-017

- **Source:** `suites/go-core/tests/gateway_users_test.go`
- **Test:** `TestAPIToken_MeEndpoint`
- **Tags:** @svc_gateway

**Scenario:** TestAPIToken_MeEndpoint

- **Given** A valid API token exists for a user.
- **When** The token is used on the `/me` endpoint.
- **Then** The gateway authenticates the token and returns the user context.

### E2E-GO-CORE-018

- **Source:** `suites/go-core/tests/gateway_users_test.go`
- **Test:** `TestAPIToken_MeEndpointInvalidToken`
- **Tags:** @svc_gateway

**Scenario:** TestAPIToken_MeEndpointInvalidToken

- **Given** An invalid API token is supplied.
- **When** The token is used on the `/me` endpoint.
- **Then** The gateway rejects the request as unauthenticated.

### E2E-GO-CORE-019

- **Source:** `suites/go-core/tests/gateway_users_test.go`
- **Test:** `TestAPIToken_ConnectRPCEndpointAuthenticated`
- **Tags:** @svc_gateway

**Scenario:** TestAPIToken_ConnectRPCEndpointAuthenticated

- **Given** A valid API token exists for a user.
- **When** The token is used on a ConnectRPC endpoint.
- **Then** The gateway authenticates the token and allows the request.

### E2E-GO-CORE-020

- **Source:** `suites/go-core/tests/gateway_users_test.go`
- **Test:** `TestAPIToken_ConnectRPCEndpointInvalidToken`
- **Tags:** @svc_gateway

**Scenario:** TestAPIToken_ConnectRPCEndpointInvalidToken

- **Given** An invalid API token is supplied.
- **When** The token is used on a ConnectRPC endpoint.
- **Then** The gateway rejects the request.

### E2E-GO-CORE-021

- **Source:** `suites/go-core/tests/gateway_users_test.go`
- **Test:** `TestUsersGateway_CreateAndRevokeAPIToken`
- **Tags:** @svc_gateway, @svc_identity

**Scenario:** TestUsersGateway_CreateAndRevokeAPIToken

- **Given** An authenticated user can manage API tokens.
- **When** The user creates an API token and then revokes it.
- **Then** The token is created, visible as expected, and no longer usable after revocation.

### E2E-GO-CORE-022

- **Source:** `suites/go-core/tests/gateway_users_test.go`
- **Test:** `TestUsersGateway_ListAPITokens`
- **Tags:** @svc_gateway, @svc_identity

**Scenario:** TestUsersGateway_ListAPITokens

- **Given** A user has API tokens.
- **When** The user lists API tokens through the gateway.
- **Then** The gateway returns the token records available to that user.

### E2E-GO-CORE-023

- **Source:** `suites/go-core/tests/gateway_users_test.go`
- **Test:** `TestUsersGateway_RevokeAPITokenNotFound`
- **Tags:** @svc_gateway, @svc_identity

**Scenario:** TestUsersGateway_RevokeAPITokenNotFound

- **Given** A user requests revocation for a nonexistent token.
- **When** The revoke API is called.
- **Then** The gateway returns a not-found response.

### E2E-GO-CORE-024

- **Source:** `suites/go-core/tests/gateway_users_test.go`
- **Test:** `TestUsersGateway_CreateAPITokenUnauthenticated`
- **Tags:** @svc_gateway, @svc_identity

**Scenario:** TestUsersGateway_CreateAPITokenUnauthenticated

- **Given** No user credentials are supplied.
- **When** A caller requests API-token creation.
- **Then** The gateway rejects the request as unauthenticated.

### E2E-GO-CORE-025

- **Source:** `suites/go-core/tests/gateway_users_test.go`
- **Test:** `TestAPIToken_CreatedTokenAuthenticates`
- **Tags:** @svc_gateway, @svc_identity

**Scenario:** TestAPIToken_CreatedTokenAuthenticates

- **Given** A user creates a new API token.
- **When** The new token is used for an authenticated request.
- **Then** The gateway accepts the token and identifies the user.

### E2E-GO-CORE-026

- **Source:** `suites/go-core/tests/gateway_ziti_test.go`
- **Test:** `TestZitiMeEndpointAuthenticated`
- **Tags:** @svc_gateway

**Scenario:** TestZitiMeEndpointAuthenticated

- **Given** A Ziti-authenticated request reaches the gateway.
- **When** The caller requests the `/me` endpoint.
- **Then** The gateway returns the authenticated Ziti user context.

### E2E-GO-CORE-027

- **Source:** `suites/go-core/tests/idle_test.go`
- **Test:** `TestWorkloadStopsAfterIdleTimeout`
- **Tags:** @svc_runners, @svc_k8s_runner

**Scenario:** TestWorkloadStopsAfterIdleTimeout

- **Given** A workload is started with an idle-timeout policy.
- **When** The workload remains idle past the configured timeout.
- **Then** The runner stops the workload automatically.

### E2E-GO-CORE-028

- **Source:** `suites/go-core/tests/imagepull_test.go`
- **Test:** `TestImagePullSecretAttachedToPod`
- **Tags:** @svc_runners, @svc_k8s_runner

**Scenario:** TestImagePullSecretAttachedToPod

- **Given** An agent or workload is configured with an image pull secret.
- **When** A pod is created for that workload.
- **Then** The resulting pod includes the configured image pull secret.

### E2E-GO-CORE-029

- **Source:** `suites/go-core/tests/k8s_runner_errors_test.go`
- **Test:** `TestErrors/start_workload_missing_image`
- **Tags:** @svc_k8s_runner

**Scenario:** TestErrors/start_workload_missing_image

- **Given** A runner receives a start-workload request without an image.
- **When** The runner validates the request.
- **Then** The runner returns a clear error instead of creating a workload.

### E2E-GO-CORE-030

- **Source:** `suites/go-core/tests/k8s_runner_errors_test.go`
- **Test:** `TestErrors/inspect_nonexistent_workload`
- **Tags:** @svc_k8s_runner

**Scenario:** TestErrors/inspect_nonexistent_workload

- **Given** No workload exists for the requested id.
- **When** A client inspects that workload.
- **Then** The runner returns a not-found error.

### E2E-GO-CORE-031

- **Source:** `suites/go-core/tests/k8s_runner_errors_test.go`
- **Test:** `TestErrors/stream_logs_missing_container_name`
- **Tags:** @svc_k8s_runner

**Scenario:** TestErrors/stream_logs_missing_container_name

- **Given** A log-stream request omits the container name.
- **When** The runner validates the request.
- **Then** The runner returns a validation error.

### E2E-GO-CORE-032

- **Source:** `suites/go-core/tests/k8s_runner_errors_test.go`
- **Test:** `TestErrors/exec_on_nonexistent_workload`
- **Tags:** @svc_k8s_runner

**Scenario:** TestErrors/exec_on_nonexistent_workload

- **Given** No workload exists for the requested id.
- **When** A client attempts to execute a command in that workload.
- **Then** The runner returns a not-found error.

### E2E-GO-CORE-033

- **Source:** `suites/go-core/tests/k8s_runner_errors_test.go`
- **Test:** `TestErrors/remove_nonexistent_volume`
- **Tags:** @svc_k8s_runner

**Scenario:** TestErrors/remove_nonexistent_volume

- **Given** No volume exists for the requested id.
- **When** A client requests volume removal.
- **Then** The runner returns a not-found error.

### E2E-GO-CORE-034

- **Source:** `suites/go-core/tests/k8s_runner_exec_test.go`
- **Test:** `TestExec/basic_command`
- **Tags:** @svc_k8s_runner

**Scenario:** TestExec/basic_command

- **Given** A workload is running.
- **When** A client executes a basic command inside it.
- **Then** The runner returns the command output and a zero exit code.

### E2E-GO-CORE-035

- **Source:** `suites/go-core/tests/k8s_runner_exec_test.go`
- **Test:** `TestExec/shell_command`
- **Tags:** @svc_k8s_runner

**Scenario:** TestExec/shell_command

- **Given** A workload is running with shell support.
- **When** A client executes a shell command inside it.
- **Then** The runner returns the shell command output.

### E2E-GO-CORE-036

- **Source:** `suites/go-core/tests/k8s_runner_exec_test.go`
- **Test:** `TestExec/nonzero_exit_code`
- **Tags:** @svc_k8s_runner

**Scenario:** TestExec/nonzero_exit_code

- **Given** A workload is running.
- **When** A client executes a command that exits nonzero.
- **Then** The runner reports the nonzero exit code and output.

### E2E-GO-CORE-037

- **Source:** `suites/go-core/tests/k8s_runner_exec_test.go`
- **Test:** `TestExec/stdin_and_eof`
- **Tags:** @svc_k8s_runner

**Scenario:** TestExec/stdin_and_eof

- **Given** A workload is running.
- **When** A client executes a command that reads standard input until EOF.
- **Then** The runner delivers stdin and completes the command when EOF is sent.

### E2E-GO-CORE-038

- **Source:** `suites/go-core/tests/k8s_runner_exec_test.go`
- **Test:** `TestExec/cancel_execution`
- **Tags:** @svc_k8s_runner

**Scenario:** TestExec/cancel_execution

- **Given** A long-running command is executing in a workload.
- **When** The client cancels the execution.
- **Then** The runner stops streaming and reports cancellation behavior.

### E2E-GO-CORE-039

- **Source:** `suites/go-core/tests/k8s_runner_exec_test.go`
- **Test:** `TestExec/workdir_and_env`
- **Tags:** @svc_k8s_runner

**Scenario:** TestExec/workdir_and_env

- **Given** A workload supports command environment and working directory settings.
- **When** A client executes a command with env vars and workdir.
- **Then** The command observes the requested environment and working directory.

### E2E-GO-CORE-040

- **Source:** `suites/go-core/tests/k8s_runner_storage_test.go`
- **Test:** `TestPutArchive`
- **Tags:** @svc_k8s_runner

**Scenario:** TestPutArchive

- **Given** A workload is running with writable storage.
- **When** A client uploads an archive into the workload.
- **Then** The archive contents are available in the workload filesystem.

### E2E-GO-CORE-041

- **Source:** `suites/go-core/tests/k8s_runner_streaming_test.go`
- **Test:** `TestStreaming/logs_follow`
- **Tags:** @svc_k8s_runner

**Scenario:** TestStreaming/logs_follow

- **Given** A workload produces logs over time.
- **When** A client streams logs with follow enabled.
- **Then** The stream includes new log lines as they are produced.

### E2E-GO-CORE-042

- **Source:** `suites/go-core/tests/k8s_runner_streaming_test.go`
- **Test:** `TestStreaming/logs_tail`
- **Tags:** @svc_k8s_runner

**Scenario:** TestStreaming/logs_tail

- **Given** A workload has existing logs.
- **When** A client streams logs with a tail limit.
- **Then** The stream returns only the requested tail of log output.

### E2E-GO-CORE-043

- **Source:** `suites/go-core/tests/k8s_runner_streaming_test.go`
- **Test:** `TestStreamEvents`
- **Tags:** @svc_k8s_runner

**Scenario:** TestStreamEvents

- **Given** A workload lifecycle produces events.
- **When** A client streams workload events.
- **Then** The runner emits the expected event sequence. TODO: clarify.

### E2E-GO-CORE-044

- **Source:** `suites/go-core/tests/k8s_runner_volume_test.go`
- **Test:** `TestVolumeQueries/list_workloads_by_volume`
- **Tags:** @svc_k8s_runner

**Scenario:** TestVolumeQueries/list_workloads_by_volume

- **Given** Workloads are attached to a volume.
- **When** A client lists workloads by volume.
- **Then** The runner returns the workloads associated with that volume.

### E2E-GO-CORE-045

- **Source:** `suites/go-core/tests/k8s_runner_volume_test.go`
- **Test:** `TestVolumeQueries/remove_volume`
- **Tags:** @svc_k8s_runner

**Scenario:** TestVolumeQueries/remove_volume

- **Given** A volume exists in the runner.
- **When** A client removes the volume.
- **Then** The volume is deleted and subsequent queries reflect removal.

### E2E-GO-CORE-046

- **Source:** `suites/go-core/tests/k8s_runner_workload_test.go`
- **Test:** `TestReady`
- **Tags:** @svc_k8s_runner, @smoke

**Scenario:** TestReady

- **Given** The k8s runner service is deployed.
- **When** A client calls the readiness endpoint.
- **Then** The runner reports ready.

### E2E-GO-CORE-047

- **Source:** `suites/go-core/tests/k8s_runner_workload_test.go`
- **Test:** `TestWorkloadLifecycle/start_and_inspect`
- **Tags:** @svc_k8s_runner

**Scenario:** TestWorkloadLifecycle/start_and_inspect

- **Given** A valid workload spec is available.
- **When** A client starts and inspects the workload.
- **Then** The workload reaches a running state and inspection returns its metadata.

### E2E-GO-CORE-048

- **Source:** `suites/go-core/tests/k8s_runner_workload_test.go`
- **Test:** `TestWorkloadLifecycle/start_with_env_and_workdir`
- **Tags:** @svc_k8s_runner

**Scenario:** TestWorkloadLifecycle/start_with_env_and_workdir

- **Given** A workload spec includes env vars and a working directory.
- **When** A client starts the workload.
- **Then** The workload is created with the requested environment and working directory.

### E2E-GO-CORE-049

- **Source:** `suites/go-core/tests/k8s_runner_workload_test.go`
- **Test:** `TestWorkloadLifecycle/start_with_sidecars`
- **Tags:** @svc_k8s_runner

**Scenario:** TestWorkloadLifecycle/start_with_sidecars

- **Given** A workload spec includes sidecar containers.
- **When** A client starts the workload.
- **Then** The workload includes the requested sidecars.

### E2E-GO-CORE-050

- **Source:** `suites/go-core/tests/k8s_runner_workload_test.go`
- **Test:** `TestWorkloadLifecycle/start_with_custom_labels`
- **Tags:** @svc_k8s_runner

**Scenario:** TestWorkloadLifecycle/start_with_custom_labels

- **Given** A workload spec includes custom labels.
- **When** A client starts the workload.
- **Then** The workload metadata contains the custom labels.

### E2E-GO-CORE-051

- **Source:** `suites/go-core/tests/k8s_runner_workload_test.go`
- **Test:** `TestWorkloadLifecycle/touch_workload`
- **Tags:** @svc_k8s_runner

**Scenario:** TestWorkloadLifecycle/touch_workload

- **Given** A workload is running.
- **When** A client touches the workload.
- **Then** The workload remains active and its activity timestamp is updated. TODO: clarify.

### E2E-GO-CORE-052

- **Source:** `suites/go-core/tests/k8s_runner_workload_test.go`
- **Test:** `TestWorkloadLifecycle/stop_workload`
- **Tags:** @svc_k8s_runner

**Scenario:** TestWorkloadLifecycle/stop_workload

- **Given** A workload is running.
- **When** A client stops the workload.
- **Then** The workload transitions to stopped.

### E2E-GO-CORE-053

- **Source:** `suites/go-core/tests/k8s_runner_workload_test.go`
- **Test:** `TestWorkloadLifecycle/remove_workload_with_volumes`
- **Tags:** @svc_k8s_runner

**Scenario:** TestWorkloadLifecycle/remove_workload_with_volumes

- **Given** A workload with attached volumes exists.
- **When** A client removes the workload.
- **Then** The workload and associated runner resources are removed.

### E2E-GO-CORE-054

- **Source:** `suites/go-core/tests/llm_proxy_test.go`
- **Test:** `TestLLMProxyURLDefaultUsesIngress`
- **Tags:** @svc_llm_proxy

**Scenario:** TestLLMProxyURLDefaultUsesIngress

- **Given** Ingress domain and port environment values are set.
- **When** The LLM proxy URL is built without an explicit override.
- **Then** The URL uses the ingress host and port.

### E2E-GO-CORE-055

- **Source:** `suites/go-core/tests/llm_proxy_test.go`
- **Test:** `TestLLMProxyURLExplicitOverride`
- **Tags:** @svc_llm_proxy

**Scenario:** TestLLMProxyURLExplicitOverride

- **Given** An explicit LLM proxy URL is configured.
- **When** The LLM proxy URL is requested.
- **Then** The configured URL is used unchanged.

### E2E-GO-CORE-056

- **Source:** `suites/go-core/tests/llm_proxy_test.go`
- **Test:** `TestLLMGatewayConnectEndpointPath`
- **Tags:** @svc_llm, @svc_llm_proxy

**Scenario:** TestLLMGatewayConnectEndpointPath

- **Given** The LLM gateway Connect endpoint is addressed.
- **When** A client builds the ConnectRPC path.
- **Then** The path matches the expected gateway service route.

### E2E-GO-CORE-057

- **Source:** `suites/go-core/tests/llm_proxy_test.go`
- **Test:** `TestLLMProxyGatewayCreatedModel`
- **Tags:** @svc_llm, @svc_llm_proxy

**Scenario:** TestLLMProxyGatewayCreatedModel

- **Given** A model is created through the gateway for LLM proxy use.
- **When** A request is sent through the LLM proxy for that model.
- **Then** The proxy routes to the configured model endpoint and returns a response.

### E2E-GO-CORE-058

- **Source:** `suites/go-core/tests/mcp_test.go`
- **Test:** `TestMCPToolsE2E`
- **Tags:** @svc_agents_orchestrator

**Scenario:** TestMCPToolsE2E

- **Given** An agent has MCP tools configured.
- **When** A user sends a prompt requiring tool use.
- **Then** The agent invokes the MCP tools and returns the expected answer.

### E2E-GO-CORE-059

- **Source:** `suites/go-core/tests/mcp_test.go`
- **Test:** `TestMCPToolsAgnE2E`
- **Tags:** @svc_agents_orchestrator

**Scenario:** TestMCPToolsAgnE2E

- **Given** An AGN-backed agent has MCP tools configured.
- **When** A user sends a prompt requiring tool use.
- **Then** The AGN agent invokes the MCP tools and returns the expected answer.

### E2E-GO-CORE-060

- **Source:** `suites/go-core/tests/media_proxy_external_test.go`
- **Test:** `TestExternalProxy_PublicImage`
- **Tags:** @svc_media_proxy

**Scenario:** TestExternalProxy_PublicImage

- **Given** A public image URL is available.
- **When** A caller requests it through the media proxy.
- **Then** The proxy returns the image content.

### E2E-GO-CORE-061

- **Source:** `suites/go-core/tests/media_proxy_external_test.go`
- **Test:** `TestExternalProxy_PublicImageWithResize`
- **Tags:** @svc_media_proxy

**Scenario:** TestExternalProxy_PublicImageWithResize

- **Given** A public image URL is available and resize params are supplied.
- **When** A caller requests it through the media proxy.
- **Then** The proxy returns resized image content.

### E2E-GO-CORE-062

- **Source:** `suites/go-core/tests/media_proxy_external_test.go`
- **Test:** `TestExternalProxy_NonImageURL`
- **Tags:** @svc_media_proxy

**Scenario:** TestExternalProxy_NonImageURL

- **Given** A public URL points to non-image content.
- **When** A caller requests it through the media proxy.
- **Then** The proxy rejects or refuses to proxy the non-image content.

### E2E-GO-CORE-063

- **Source:** `suites/go-core/tests/media_proxy_external_test.go`
- **Test:** `TestExternalProxy_Unauthenticated`
- **Tags:** @svc_media_proxy

**Scenario:** TestExternalProxy_Unauthenticated

- **Given** No credentials are supplied.
- **When** A caller requests an external media proxy URL.
- **Then** The media proxy rejects the request as unauthenticated.

### E2E-GO-CORE-064

- **Source:** `suites/go-core/tests/media_proxy_external_test.go`
- **Test:** `TestExternalProxy_InvalidToken`
- **Tags:** @svc_media_proxy

**Scenario:** TestExternalProxy_InvalidToken

- **Given** An invalid token is supplied.
- **When** A caller requests an external media proxy URL.
- **Then** The media proxy rejects the request.

### E2E-GO-CORE-065

- **Source:** `suites/go-core/tests/media_proxy_external_test.go`
- **Test:** `TestExternalProxy_MissingURL`
- **Tags:** @svc_media_proxy

**Scenario:** TestExternalProxy_MissingURL

- **Given** No upstream URL is supplied.
- **When** A caller requests the external media proxy endpoint.
- **Then** The media proxy returns a validation error.

### E2E-GO-CORE-066

- **Source:** `suites/go-core/tests/media_proxy_file_test.go`
- **Test:** `TestFileProxy_UploadAndProxy`
- **Tags:** @svc_media_proxy, @svc_files

**Scenario:** TestFileProxy_UploadAndProxy

- **Given** A file is uploaded to the files service.
- **When** A caller requests it through the media proxy.
- **Then** The proxy returns the uploaded file content.

### E2E-GO-CORE-067

- **Source:** `suites/go-core/tests/media_proxy_file_test.go`
- **Test:** `TestFileProxy_UploadAndProxyWithResize`
- **Tags:** @svc_media_proxy, @svc_files

**Scenario:** TestFileProxy_UploadAndProxyWithResize

- **Given** An uploaded image file exists and resize params are supplied.
- **When** A caller requests it through the media proxy.
- **Then** The proxy returns resized image content.

### E2E-GO-CORE-068

- **Source:** `suites/go-core/tests/media_proxy_file_test.go`
- **Test:** `TestFileProxy_NotFoundFile`
- **Tags:** @svc_media_proxy, @svc_files

**Scenario:** TestFileProxy_NotFoundFile

- **Given** No file exists for the requested id.
- **When** A caller requests it through the media proxy.
- **Then** The media proxy returns a not-found response.

### E2E-GO-CORE-069

- **Source:** `suites/go-core/tests/media_proxy_file_test.go`
- **Test:** `TestFileProxy_RangeRequest`
- **Tags:** @svc_media_proxy, @svc_files

**Scenario:** TestFileProxy_RangeRequest

- **Given** An uploaded file exists.
- **When** A caller requests a byte range through the media proxy.
- **Then** The proxy returns the requested range with appropriate range semantics.

### E2E-GO-CORE-070

- **Source:** `suites/go-core/tests/media_proxy_file_test.go`
- **Test:** `TestFileProxy_Unauthenticated`
- **Tags:** @svc_media_proxy, @svc_files

**Scenario:** TestFileProxy_Unauthenticated

- **Given** No credentials are supplied.
- **When** A caller requests a file through the media proxy.
- **Then** The media proxy rejects the request as unauthenticated.

### E2E-GO-CORE-071

- **Source:** `suites/go-core/tests/metering_test.go`
- **Test:** `TestRecordAndQueryUsage`
- **Tags:** @svc_metering

**Scenario:** TestRecordAndQueryUsage

- **Given** Usage can be recorded for metered resources.
- **When** The test records usage and queries it back.
- **Then** The metering service returns the recorded usage values.

### E2E-GO-CORE-072

- **Source:** `suites/go-core/tests/multi_test.go`
- **Test:** `TestMultipleAgentsSeparateThreads`
- **Tags:** @svc_agents_orchestrator

**Scenario:** TestMultipleAgentsSeparateThreads

- **Given** Multiple agents and threads exist.
- **When** Messages are sent to separate agents on separate threads.
- **Then** Each agent responds in the correct thread without cross-talk.

### E2E-GO-CORE-073

- **Source:** `suites/go-core/tests/multi_test.go`
- **Test:** `TestSameAgentMultipleThreads`
- **Tags:** @svc_agents_orchestrator

**Scenario:** TestSameAgentMultipleThreads

- **Given** One agent participates in multiple threads.
- **When** Messages are sent on multiple threads.
- **Then** The agent responds in each thread while preserving thread separation.

### E2E-GO-CORE-074

- **Source:** `suites/go-core/tests/organizations_test.go`
- **Test:** `TestOrganizationsServiceE2E`
- **Tags:** @svc_organizations

**Scenario:** TestOrganizationsServiceE2E

- **Given** Organization APIs are available.
- **When** A client creates, reads, updates, lists, and deletes organization data.
- **Then** Organization state changes are visible through the service APIs.

### E2E-GO-CORE-075

- **Source:** `suites/go-core/tests/pipeline_test.go`
- **Test:** `TestFullPipelineMessageResponse`
- **Tags:** @svc_agents_orchestrator, @svc_llm, @smoke

**Scenario:** TestFullPipelineMessageResponse

- **Given** A full agent pipeline is configured.
- **When** A user sends a message to the agent.
- **Then** The pipeline produces the expected response in the thread.

### E2E-GO-CORE-082

- **Source:** `suites/go-core/tests/tracing_test.go`
- **Test:** `TestAgentSimpleHelloProducesTrace`
- **Tags:** @svc_agents_orchestrator

**Scenario:** TestAgentSimpleHelloProducesTrace

- **Given** Tracing is available and an agent can answer a simple message.
- **When** A simple hello pipeline runs.
- **Then** The trace contains invocation and LLM spans with the expected response attributes.

### E2E-GO-CORE-083

- **Source:** `suites/go-core/tests/tracing_test.go`
- **Test:** `TestAgentMCPToolsProducesTrace`
- **Tags:** @svc_agents_orchestrator

**Scenario:** TestAgentMCPToolsProducesTrace

- **Given** Tracing is available and an agent uses MCP tools.
- **When** A tool-using pipeline runs.
- **Then** The trace contains invocation, LLM, and tool execution spans for the expected tools.

### E2E-GO-CORE-084

- **Source:** `suites/go-core/tests/workload_start_retry_policy_test.go`
- **Test:** `TestWorkloadStartRetryPolicyFastRetry`
- **Tags:** @svc_agents_orchestrator, @svc_runners, @svc_k8s_runner

**Scenario:** TestWorkloadStartRetryPolicyFastRetry

- **Given** A workload start fails because of invalid container configuration.
- **When** The orchestrator observes failure and retries.
- **Then** Failed workloads are recorded, retry happens quickly, the thread stays active, and a later valid workload produces the expected response.

### E2E-GO-CORE-087

- **Source:** `suites/go-core/tests/pipeline_test.go`
- **Test:** `TestFullPipelineAgnMessageResponse`
- **Tags:** @svc_agents_orchestrator, @svc_llm

**Scenario:** TestFullPipelineAgnMessageResponse

- **Given** A full AGN-backed agent pipeline is configured.
- **When** A user sends a message to the agent.
- **Then** The pipeline produces the expected response in the thread.
- **And** A real workload pod exists and its Ziti enrollment, gateway wait, and restartable sidecar containers are healthy.

### E2E-GO-CORE-088

- **Source:** `suites/go-core/tests/pipeline_test.go`
- **Test:** `TestFullPipelineClaudeMessageResponse`
- **Tags:** @svc_agents_orchestrator, @svc_llm

**Scenario:** TestFullPipelineClaudeMessageResponse

- **Given** A full Claude-backed agent pipeline is configured.
- **When** A user sends a message to the agent.
- **Then** The pipeline produces the expected response in the thread.

### E2E-GO-CORE-089

- **Source:** `suites/go-core/tests/runners_sampled_at_test.go`
- **Test:** `TestBatchUpdateWorkloadSampledAtSingle`
- **Tags:** @svc_runners

**Scenario:** TestBatchUpdateWorkloadSampledAtSingle

- **Given** A single workload has a sampled-at observation update.
- **When** The batch update operation runs.
- **Then** The workload sampled-at value is updated.

### E2E-GO-CORE-090

- **Source:** `suites/go-core/tests/runners_sampled_at_test.go`
- **Test:** `TestBatchUpdateWorkloadSampledAtMultiple`
- **Tags:** @svc_runners

**Scenario:** TestBatchUpdateWorkloadSampledAtMultiple

- **Given** Multiple workloads have sampled-at observation updates.
- **When** The batch update operation runs.
- **Then** Each workload sampled-at value is updated.

### E2E-GO-CORE-091

- **Source:** `suites/go-core/tests/runners_sampled_at_test.go`
- **Test:** `TestBatchUpdateWorkloadSampledAtIdempotent`
- **Tags:** @svc_runners

**Scenario:** TestBatchUpdateWorkloadSampledAtIdempotent

- **Given** A workload sampled-at update has already been applied.
- **When** The same batch update is applied again.
- **Then** The operation is idempotent and keeps the expected sampled-at value.

### E2E-GO-CORE-092

- **Source:** `suites/go-core/tests/runners_sampled_at_test.go`
- **Test:** `TestBatchUpdateVolumeSampledAtSingle`
- **Tags:** @svc_runners

**Scenario:** TestBatchUpdateVolumeSampledAtSingle

- **Given** A single volume has a sampled-at observation update.
- **When** The batch update operation runs.
- **Then** The volume sampled-at value is updated.

### E2E-GO-CORE-093

- **Source:** `suites/go-core/tests/runners_sampled_at_test.go`
- **Test:** `TestBatchUpdateVolumeSampledAtMultiple`
- **Tags:** @svc_runners

**Scenario:** TestBatchUpdateVolumeSampledAtMultiple

- **Given** Multiple volumes have sampled-at observation updates.
- **When** The batch update operation runs.
- **Then** Each volume sampled-at value is updated.

### E2E-GO-CORE-094

- **Source:** `suites/go-core/tests/runners_sampled_at_test.go`
- **Test:** `TestBatchUpdateVolumeSampledAtIdempotent`
- **Tags:** @svc_runners

**Scenario:** TestBatchUpdateVolumeSampledAtIdempotent

- **Given** A volume sampled-at update has already been applied.
- **When** The same batch update is applied again.
- **Then** The operation is idempotent and keeps the expected sampled-at value.

### E2E-GO-CORE-095

- **Source:** `suites/go-core/tests/runners_test.go`
- **Test:** `TestRunnerLifecycle`
- **Tags:** @svc_runners

**Scenario:** TestRunnerLifecycle

- **Given** Runner APIs are available.
- **When** A client creates, reads, updates, lists, and deletes a runner.
- **Then** Runner lifecycle state changes are visible through the service APIs.

### E2E-GO-CORE-096

- **Source:** `suites/go-core/tests/start_test.go`
- **Test:** `TestWorkloadStartsOnUnackedMessage`
- **Tags:** @svc_agents_orchestrator, @svc_runners

**Scenario:** TestWorkloadStartsOnUnackedMessage

- **Given** A message is unacknowledged for an agent.
- **When** The orchestrator evaluates work that needs to start.
- **Then** A workload starts to process the unacknowledged message.

### E2E-GO-CORE-097

- **Source:** `suites/go-core/tests/threads_send_test.go`
- **Test:** `TestThreadsSendShell`
- **Tags:** @svc_agents_orchestrator

**Scenario:** TestThreadsSendShell

- **Given** A shell-capable thread send path is available.
- **When** A user sends a shell message.
- **Then** The thread send operation creates or updates the thread and records the shell message. TODO: clarify.

### E2E-GO-CORE-098

- **Source:** `suites/go-core/tests/egress_controlplane_test.go`
- **Test:** `TestEgressGatewayFeaturePath`
- **Tags:** @svc_egress

**Scenario:** TestEgressGatewayFeaturePath

- **Given** An organization, authorized user identity, generated agent identity, and secret-backed allow egress rule fixture.
- **When** The allow rule is created, attached to the agent, and listed through the gateway lookup path.
- **Then** The egress service returns the attached allow rule for the agent.
- **And** Secrets refuses deletion while the secret is referenced by the egress rule.
- **And** The secret value can still be resolved through the Secrets integration.
- **And** The rule configuration preserves the secret-backed injected header and attachment metadata.

### E2E-GO-CORE-099

- **Source:** `suites/go-core/tests/egress_controlplane_test.go`
- **Test:** `TestEgressGatewayDenyAndNoRulePaths`
- **Tags:** @svc_egress

**Scenario:** TestEgressGatewayDenyAndNoRulePaths

- **Given** An authorized agent starts without any egress rule attachment.
- **When** The gateway lookup path lists rules for the unattached agent.
- **Then** No egress rules are returned.
- **When** A deny egress rule is created and attached to the same agent.
- **Then** The gateway lookup path returns the deny rule for that agent.
- **And** The returned rule effect is `DENY`.

### E2E-GO-CORE-100

- **Source:** `suites/go-core/tests/egress_gateway_wiring_test.go`
- **Test:** `TestEgressGatewayDeploymentWiring`
- **Tags:** @svc_egress_gateway

**Scenario:** TestEgressGatewayDeploymentWiring

- **Given** The platform installation includes egress gateway and k8s-runner workload wiring.
- **When** The suite inspects the gateway deployment, health endpoint, CA/Ziti identity secrets, workload NetworkPolicy, and inline CA workload path.
- **Then** The gateway exposes the expected health/admin wiring and required data-plane environment/mount configuration.
- **And** The egress CA and Ziti identity/enrollment secrets exist with required data.
- **And** The managed workload NetworkPolicy selects Agyn-managed pods, includes Egress policy type, allows configured pod, service, and additional internal CIDR exclusions, allows OpenZiti CIDR, and allows DNS over TCP/UDP 53.
- **And** The public internet egress rule allows `0.0.0.0/0` with blocked CIDR exceptions for private, link-local, loopback, configured cluster pod, configured cluster service, and configured additional internal CIDRs.
- **And** A k8s-runner workload can receive and use the egress CA via inline file/env contract.

### E2E-GO-CORE-101

- **Source:** `suites/go-core/tests/egress_dataplane_test.go`
- **Test:** `TestEgressGatewayDataPlaneHTTPBehavior`
- **Tags:** @svc_egress_gateway

**Scenario:** TestEgressGatewayDataPlaneHTTPBehavior

- **Given** A live egress data-plane endpoint and distinct direct bypass endpoint are configured.
- **When** Requests are sent through allow, deny, literal-header, secret-header, no-match direct bypass, and websocket upgrade paths.
- **Then** Allowed requests succeed, denied requests are forbidden, unmatched traffic bypasses gateway injection, websocket requests require upgrade, and literal plus Secret-backed injected header markers are present.

### E2E-GO-CORE-102

- **Source:** `suites/go-core/tests/egress_dataplane_test.go`
- **Test:** `TestEgressGatewayDataPlaneMatcherMatrix`
- **Tags:** @svc_egress_gateway

**Scenario:** TestEgressGatewayDataPlaneMatcherMatrix

- **Given** A live egress data-plane endpoint is configured for matcher validation.
- **When** Requests are sent through the gateway with path pattern `/repos/**`, method, and port-specific fixtures.
- **Then** Matching requests return success and expose the expected path, method, or port marker header.
- **And** A `/repos` request that does not satisfy the `/repos/**` boundary returns a no-match response.
