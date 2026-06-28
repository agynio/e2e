# Go AGN CLI

- **Suite ID:** E2E-SUITE-GO-AGN-CLI
- **Source:** `suites/go-agn-cli/suite.yaml`
- **Kind:** Go
- **Tags:** @svc_agn_cli

## Intent

Exercises the released or supplied `agn` CLI against the platform, focusing on command execution, session persistence, resume behavior, summarization, tool-pair summarization, and system-prompt behavior.

## Scope

- Source directory: `suites/go-agn-cli`
- Test inventory pattern: `tests/*_test.go`
- Included case count: 6

## Actors

- AGN CLI user
- Agent runtime
- LLM endpoint
- MCP weather test server

## Preconditions

- An `agn` binary is available on `PATH` or supplied to the suite runner.
- The platform is reachable with credentials and agent-init images configured.
- Required LLM and MCP test endpoints are reachable.

## Tags

- `@svc_agn_cli`

## Case index

| Case ID | Source test | Tags |
| --- | --- | --- |
| [E2E-GO-AGN-CLI-001](#e2e-go-agn-cli-001) | `TestAgnExecHello` | @svc_agn_cli |
| [E2E-GO-AGN-CLI-002](#e2e-go-agn-cli-002) | `TestExecStatePersistence` | @svc_agn_cli |
| [E2E-GO-AGN-CLI-003](#e2e-go-agn-cli-003) | `TestExecResume` | @svc_agn_cli |
| [E2E-GO-AGN-CLI-004](#e2e-go-agn-cli-004) | `TestAgentSystemPrompt` | @svc_agn_cli |
| [E2E-GO-AGN-CLI-005](#e2e-go-agn-cli-005) | `TestSummarization` | @svc_agn_cli |
| [E2E-GO-AGN-CLI-006](#e2e-go-agn-cli-006) | `TestSummarizationToolPair` | @svc_agn_cli |

## Scenarios

### E2E-GO-AGN-CLI-001

- **Source:** `suites/go-agn-cli/tests/main_test.go`
- **Test:** `TestAgnExecHello`
- **Tags:** @svc_agn_cli

**Scenario:** TestAgnExecHello

- **Given** AGN CLI execution is configured against the platform.
- **When** A user runs a simple hello command through `agn exec`.
- **Then** The command completes and returns the expected agent response.

### E2E-GO-AGN-CLI-002

- **Source:** `suites/go-agn-cli/tests/main_test.go`
- **Test:** `TestExecStatePersistence`
- **Tags:** @svc_agn_cli

**Scenario:** TestExecStatePersistence

- **Given** An AGN CLI execution can persist local session state.
- **When** A user runs a command that writes state and then inspects that state in a later execution.
- **Then** The later execution observes the persisted state from the prior run.

### E2E-GO-AGN-CLI-003

- **Source:** `suites/go-agn-cli/tests/main_test.go`
- **Test:** `TestExecResume`
- **Tags:** @svc_agn_cli

**Scenario:** TestExecResume

- **Given** A previous AGN CLI execution exists and can be resumed.
- **When** A user resumes the execution and sends another prompt.
- **Then** The resumed execution continues with the existing context instead of starting from scratch.

### E2E-GO-AGN-CLI-004

- **Source:** `suites/go-agn-cli/tests/agent_system_prompt_test.go`
- **Test:** `TestAgentSystemPrompt`
- **Tags:** @svc_agn_cli

**Scenario:** TestAgentSystemPrompt

- **Given** An agent is configured with a system prompt.
- **When** A user sends a prompt whose answer depends on the configured system prompt.
- **Then** The agent response reflects the system-prompt instructions.

### E2E-GO-AGN-CLI-005

- **Source:** `suites/go-agn-cli/tests/agent_summarization_test.go`
- **Test:** `TestSummarization`
- **Tags:** @svc_agn_cli

**Scenario:** TestSummarization

- **Given** An AGN CLI session contains enough conversation to require summarization.
- **When** The user continues the session beyond the summarization threshold.
- **Then** The session remains usable and preserves the relevant context after summarization.

### E2E-GO-AGN-CLI-006

- **Source:** `suites/go-agn-cli/tests/agent_summarize_tool_pair_test.go`
- **Test:** `TestSummarizationToolPair`
- **Tags:** @svc_agn_cli

**Scenario:** TestSummarizationToolPair

- **Given** An AGN CLI session includes tool-call and tool-result pairs.
- **When** The session is summarized after tool use.
- **Then** The summary keeps tool-call/tool-result relationships coherent for continued conversation.
