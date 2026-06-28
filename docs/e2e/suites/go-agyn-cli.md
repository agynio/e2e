# Go Agyn CLI

- **Suite ID:** E2E-SUITE-GO-AGYN-CLI
- **Source:** `suites/go-agyn-cli/suite.yaml`
- **Kind:** Go
- **Tags:** @svc_agyn_cli

## Intent

Exercises the released or supplied `agyn` platform CLI against the live
Gateway for platform resource commands that do not belong to the `agn`
agent-loop CLI.

## Scope

- Source directory: `suites/go-agyn-cli`
- Test inventory pattern: `tests/*_test.go`
- Included case count: 1

## Actors

- Agyn platform CLI user
- Gateway
- Egress service

## Preconditions

- An `agyn` binary is supplied to the suite runner through `AGYN_BINARY` or
  staged in `bin/agyn`.
- The platform is reachable with `AGYN_BASE_URL`, `AGYN_API_TOKEN`, and
  `AGYN_ORGANIZATION_ID` configured.

## Tags

- `@svc_agyn_cli`

## Case index

| Case ID | Source test | Tags |
| --- | --- | --- |
| [E2E-GO-AGYN-CLI-001](#e2e-go-agyn-cli-001) | `TestAgynEgressRuleLifecycle` | @svc_agyn_cli |

## Scenarios

### E2E-GO-AGYN-CLI-001

- **Source:** `suites/go-agyn-cli/tests/egress_test.go`
- **Test:** `TestAgynEgressRuleLifecycle`
- **Tags:** @svc_agyn_cli

**Scenario:** TestAgynEgressRuleLifecycle

- **Given** The released or supplied `agyn` CLI is configured with a live Gateway URL, API token, and organization id.
- **When** A user creates an egress rule with domain, port, method, `/repos/**` path matching, allow action, and a literal injected header.
- **Then** The CLI returns the created rule with the expected matcher and header source details.
- **When** The user lists, gets, updates, attaches, detaches, and deletes the egress rule through `agyn egress rule` commands.
- **Then** Each command succeeds against the live gateway, update changes are reflected, and the attachment targets the requested agent id before cleanup.
