# Playwright Tracing App

- **Suite ID:** E2E-SUITE-PLAYWRIGHT-TRACING-APP
- **Source:** `suites/playwright-tracing-app/suite.yaml`
- **Kind:** Playwright
- **Tags:** @svc_tracing_app, @svc_agents_orchestrator, @smoke

## Intent

Validates Tracing App deep links and gateway helper serialization for resolving messages to runs, OIDC return behavior, empty states, and rendered run timelines.

## Scope

- Source directory: `suites/playwright-tracing-app`
- Test inventory pattern: `test/e2e/*.spec.ts`
- Included case count: 6

## Actors

- Tracing App user
- Agent runtime
- Gateway API
- OIDC identity provider

## Preconditions

- The Tracing App is served at `E2E_BASE_URL`.
- OIDC sign-in and agents-orchestrator message-to-run resolution are available.
- Required agent init images and LLM test endpoints are configured for full-chain cases.

## Tags

- `@svc_tracing_app`
- `@svc_agents_orchestrator`
- `@smoke`

## Case index

| Case ID | Source test | Tags |
| --- | --- | --- |
| [E2E-PLAYWRIGHT-TRACING-APP-001](#e2e-playwright-tracing-app-001) | `CreateAgent payload sets internal availability enum` | _none_ |
| [E2E-PLAYWRIGHT-TRACING-APP-002](#e2e-playwright-tracing-app-002) | `CreateAgent ConnectRPC JSON uses protobuf enum name` | _none_ |
| [E2E-PLAYWRIGHT-TRACING-APP-003](#e2e-playwright-tracing-app-003) | `CreateAgent ConnectRPC proto bytes include availability value` | _none_ |
| [E2E-PLAYWRIGHT-TRACING-APP-004](#e2e-playwright-tracing-app-004) | `shows empty state for unknown message` | @svc_tracing_app, @smoke |
| [E2E-PLAYWRIGHT-TRACING-APP-005](#e2e-playwright-tracing-app-005) | `resolves message to run and renders timeline (agn)` | @svc_tracing_app, @svc_agents_orchestrator |
| [E2E-PLAYWRIGHT-TRACING-APP-006](#e2e-playwright-tracing-app-006) | `returns to deep link after login` | @svc_tracing_app, @svc_agents_orchestrator |

## Scenarios

### E2E-PLAYWRIGHT-TRACING-APP-001

- **Source:** `suites/playwright-tracing-app/test/e2e/gateway-api.spec.ts`
- **Test:** `CreateAgent payload sets internal availability enum`
- **Tags:** _none_

**Scenario:** CreateAgent payload sets internal availability enum

- **Given** A tracing helper builds a CreateAgent payload.
- **When** The payload is serialized.
- **Then** The internal availability enum is set.

### E2E-PLAYWRIGHT-TRACING-APP-002

- **Source:** `suites/playwright-tracing-app/test/e2e/gateway-api.spec.ts`
- **Test:** `CreateAgent ConnectRPC JSON uses protobuf enum name`
- **Tags:** _none_

**Scenario:** CreateAgent ConnectRPC JSON uses protobuf enum name

- **Given** A tracing helper builds a CreateAgent ConnectRPC JSON request.
- **When** The JSON payload is serialized.
- **Then** The availability enum uses the protobuf enum name.

### E2E-PLAYWRIGHT-TRACING-APP-003

- **Source:** `suites/playwright-tracing-app/test/e2e/gateway-api.spec.ts`
- **Test:** `CreateAgent ConnectRPC proto bytes include availability value`
- **Tags:** _none_

**Scenario:** CreateAgent ConnectRPC proto bytes include availability value

- **Given** A tracing helper builds CreateAgent protobuf bytes.
- **When** The bytes are serialized.
- **Then** The availability value is present in the encoded request.

### E2E-PLAYWRIGHT-TRACING-APP-004

- **Source:** `suites/playwright-tracing-app/test/e2e/message-deeplink-empty.spec.ts`
- **Test:** `shows empty state for unknown message`
- **Tags:** @svc_tracing_app, @smoke

**Scenario:** shows empty state for unknown message

- **Given** No run exists for an unknown message id.
- **When** The user opens a tracing message deep link for that message.
- **Then** The Tracing App shows the no-run/empty state with retry affordance.

### E2E-PLAYWRIGHT-TRACING-APP-005

- **Source:** `suites/playwright-tracing-app/test/e2e/message-deeplink-fullchain.spec.ts`
- **Test:** `resolves message to run and renders timeline (agn)`
- **Tags:** @svc_tracing_app, @svc_agents_orchestrator

**Scenario:** resolves message to run and renders timeline (agn)

- **Given** An AGN-backed agent message produces trace data.
- **When** The user opens the message deep link.
- **Then** The deep link resolves to a run page and renders the timeline.

### E2E-PLAYWRIGHT-TRACING-APP-006

- **Source:** `suites/playwright-tracing-app/test/e2e/message-deeplink-oidc.spec.ts`
- **Test:** `returns to deep link after login`
- **Tags:** @svc_tracing_app, @svc_agents_orchestrator

**Scenario:** returns to deep link after login

- **Given** The user is unauthenticated and opens a tracing message deep link.
- **When** The user completes OIDC login.
- **Then** The app returns to the original deep link and resolves the run or expected message state.
