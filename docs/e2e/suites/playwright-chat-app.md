# Playwright Chat App

- **Suite ID:** E2E-SUITE-PLAYWRIGHT-CHAT-APP
- **Source:** `suites/playwright-chat-app/suite.yaml`
- **Kind:** Playwright
- **Tags:** @svc_chat_app, @svc_gateway, @svc_agents_orchestrator, @svc_organizations, @svc_files, @svc_media_proxy, @svc_tracing_app

## Intent

Validates Chat App browser journeys and helper serialization for sign-in, chat lists, chat detail, agent replies, shared chats, status changes, file uploads, inline media, organization switching, and trace links.

## Scope

- Source directory: `suites/playwright-chat-app`
- Test inventory pattern: `test/e2e/*.spec.ts`
- Included case count: 27

## Actors

- Chat App user
- Second chat participant
- Agent runtime
- Gateway API
- Tracing App
- OIDC identity provider

## Preconditions

- The Chat App is served at `E2E_BASE_URL`.
- OIDC sign-in works for configured test users.
- Agent init images, LLM test endpoints, files, media proxy, organizations, and tracing services are available for cases that require them.

## Tags

- `@svc_chat_app`
- `@svc_gateway`
- `@svc_agents_orchestrator`
- `@svc_organizations`
- `@svc_files`
- `@svc_media_proxy`
- `@svc_tracing_app`

## Case index

| Case ID | Source test | Tags |
| --- | --- | --- |
| [E2E-PLAYWRIGHT-CHAT-APP-001](#e2e-playwright-chat-app-001) | `agent response appears after message send` | @svc_chat_app, @svc_agents_orchestrator |
| [E2E-PLAYWRIGHT-CHAT-APP-002](#e2e-playwright-chat-app-002) | `CreateAgent payload sets internal availability enum` | _none_ |
| [E2E-PLAYWRIGHT-CHAT-APP-003](#e2e-playwright-chat-app-003) | `CreateAgent ConnectRPC JSON uses protobuf enum name` | _none_ |
| [E2E-PLAYWRIGHT-CHAT-APP-004](#e2e-playwright-chat-app-004) | `CreateAgent ConnectRPC proto bytes include availability value` | _none_ |
| [E2E-PLAYWRIGHT-CHAT-APP-005](#e2e-playwright-chat-app-005) | `CreateAgent payload serializes private availability enum` | _none_ |
| [E2E-PLAYWRIGHT-CHAT-APP-006](#e2e-playwright-chat-app-006) | `CreateAgent ConnectRPC JSON uses private protobuf enum name` | _none_ |
| [E2E-PLAYWRIGHT-CHAT-APP-007](#e2e-playwright-chat-app-007) | `CreateAgent rejects unsupported availability enum` | _none_ |
| [E2E-PLAYWRIGHT-CHAT-APP-008](#e2e-playwright-chat-app-008) | `shows chat messages` | @svc_chat_app |
| [E2E-PLAYWRIGHT-CHAT-APP-009](#e2e-playwright-chat-app-009) | `two users exchange messages in a shared chat` | @svc_chat_app, @svc_gateway, @svc_organizations |
| [E2E-PLAYWRIGHT-CHAT-APP-010](#e2e-playwright-chat-app-010) | `user B sees shared chat in their chat list` | @svc_chat_app, @svc_gateway, @svc_organizations |
| [E2E-PLAYWRIGHT-CHAT-APP-011](#e2e-playwright-chat-app-011) | `switches from open to closed chat status` | @svc_chat_app |
| [E2E-PLAYWRIGHT-CHAT-APP-012](#e2e-playwright-chat-app-012) | `view trace resolves tracing deep link (${scenario.name}) - codex` | @svc_chat_app, @svc_tracing_app, @svc_agents_orchestrator, @svc_organizations |
| [E2E-PLAYWRIGHT-CHAT-APP-013](#e2e-playwright-chat-app-013) | `view trace resolves tracing deep link (${scenario.name}) - claude` | @svc_chat_app, @svc_tracing_app, @svc_agents_orchestrator, @svc_organizations |
| [E2E-PLAYWRIGHT-CHAT-APP-014](#e2e-playwright-chat-app-014) | `chat with agent and receive reply` | @svc_chat_app, @svc_agents_orchestrator |
| [E2E-PLAYWRIGHT-CHAT-APP-015](#e2e-playwright-chat-app-015) | `renders chat list on load` | @svc_chat_app |
| [E2E-PLAYWRIGHT-CHAT-APP-016](#e2e-playwright-chat-app-016) | `participant picker shows available options` | @svc_chat_app |
| [E2E-PLAYWRIGHT-CHAT-APP-017](#e2e-playwright-chat-app-017) | `redirects root to /chats` | @svc_chat_app |
| [E2E-PLAYWRIGHT-CHAT-APP-018](#e2e-playwright-chat-app-018) | `navigates to chat detail` | @svc_chat_app |
| [E2E-PLAYWRIGHT-CHAT-APP-019](#e2e-playwright-chat-app-019) | `uploads a file and renders attachment` | @svc_chat_app, @svc_files, @svc_media_proxy |
| [E2E-PLAYWRIGHT-CHAT-APP-020](#e2e-playwright-chat-app-020) | `renders mermaid diagrams inline` | @svc_chat_app, @svc_media_proxy |
| [E2E-PLAYWRIGHT-CHAT-APP-021](#e2e-playwright-chat-app-021) | `renders vega-lite charts inline` | @svc_chat_app, @svc_media_proxy |
| [E2E-PLAYWRIGHT-CHAT-APP-022](#e2e-playwright-chat-app-022) | `handles invalid mermaid input` | @svc_chat_app, @svc_media_proxy |
| [E2E-PLAYWRIGHT-CHAT-APP-023](#e2e-playwright-chat-app-023) | `handles invalid vega-lite input` | @svc_chat_app, @svc_media_proxy |
| [E2E-PLAYWRIGHT-CHAT-APP-024](#e2e-playwright-chat-app-024) | `renders multiple inline media attachments` | @svc_chat_app, @svc_media_proxy |
| [E2E-PLAYWRIGHT-CHAT-APP-025](#e2e-playwright-chat-app-025) | `switching organization updates chat list` | @svc_chat_app, @svc_organizations |
| [E2E-PLAYWRIGHT-CHAT-APP-026](#e2e-playwright-chat-app-026) | `signs in via oidc redirect flow` | @svc_chat_app |
| [E2E-PLAYWRIGHT-CHAT-APP-027](#e2e-playwright-chat-app-027) | `sign out clears oidc session storage` | @svc_chat_app |

## Scenarios

### E2E-PLAYWRIGHT-CHAT-APP-001

- **Source:** `suites/playwright-chat-app/test/e2e/chat-agent-response.spec.ts`
- **Test:** `agent response appears after message send`
- **Tags:** @svc_chat_app, @svc_agents_orchestrator

**Scenario:** agent response appears after message send

- **Given** A chat has an agent participant.
- **When** The user sends a chat message.
- **Then** An agent response appears in the conversation.

### E2E-PLAYWRIGHT-CHAT-APP-002

- **Source:** `suites/playwright-chat-app/test/e2e/chat-api.spec.ts`
- **Test:** `CreateAgent payload sets internal availability enum`
- **Tags:** _none_

**Scenario:** CreateAgent payload sets internal availability enum

- **Given** A chat helper builds a CreateAgent payload.
- **When** The payload is serialized.
- **Then** The internal availability enum is set.

### E2E-PLAYWRIGHT-CHAT-APP-003

- **Source:** `suites/playwright-chat-app/test/e2e/chat-api.spec.ts`
- **Test:** `CreateAgent ConnectRPC JSON uses protobuf enum name`
- **Tags:** _none_

**Scenario:** CreateAgent ConnectRPC JSON uses protobuf enum name

- **Given** A chat helper builds a CreateAgent ConnectRPC JSON request.
- **When** The JSON payload is serialized.
- **Then** The availability enum uses the protobuf enum name.

### E2E-PLAYWRIGHT-CHAT-APP-004

- **Source:** `suites/playwright-chat-app/test/e2e/chat-api.spec.ts`
- **Test:** `CreateAgent ConnectRPC proto bytes include availability value`
- **Tags:** _none_

**Scenario:** CreateAgent ConnectRPC proto bytes include availability value

- **Given** A chat helper builds CreateAgent protobuf bytes.
- **When** The bytes are serialized.
- **Then** The availability value is present in the encoded request.

### E2E-PLAYWRIGHT-CHAT-APP-005

- **Source:** `suites/playwright-chat-app/test/e2e/chat-api.spec.ts`
- **Test:** `CreateAgent payload serializes private availability enum`
- **Tags:** _none_

**Scenario:** CreateAgent payload serializes private availability enum

- **Given** A chat helper builds a CreateAgent payload with private availability.
- **When** The payload is serialized.
- **Then** The private availability enum is set.

### E2E-PLAYWRIGHT-CHAT-APP-006

- **Source:** `suites/playwright-chat-app/test/e2e/chat-api.spec.ts`
- **Test:** `CreateAgent ConnectRPC JSON uses private protobuf enum name`
- **Tags:** _none_

**Scenario:** CreateAgent ConnectRPC JSON uses private protobuf enum name

- **Given** A chat helper builds a private CreateAgent ConnectRPC JSON request.
- **When** The JSON payload is serialized.
- **Then** The private availability enum uses the protobuf enum name.

### E2E-PLAYWRIGHT-CHAT-APP-007

- **Source:** `suites/playwright-chat-app/test/e2e/chat-api.spec.ts`
- **Test:** `CreateAgent rejects unsupported availability enum`
- **Tags:** _none_

**Scenario:** CreateAgent rejects unsupported availability enum

- **Given** A chat helper receives an unsupported availability enum.
- **When** The payload builder validates the enum.
- **Then** The helper rejects the unsupported value.

### E2E-PLAYWRIGHT-CHAT-APP-008

- **Source:** `suites/playwright-chat-app/test/e2e/chat-detail.spec.ts`
- **Test:** `shows chat messages`
- **Tags:** @svc_chat_app

**Scenario:** shows chat messages

- **Given** A chat contains messages.
- **When** The user opens the chat detail.
- **Then** The chat messages are rendered.

### E2E-PLAYWRIGHT-CHAT-APP-009

- **Source:** `suites/playwright-chat-app/test/e2e/chat-exchange.spec.ts`
- **Test:** `two users exchange messages in a shared chat`
- **Tags:** @svc_chat_app, @svc_gateway, @svc_organizations

**Scenario:** two users exchange messages in a shared chat

- **Given** Two users share a chat.
- **When** Both users send messages in the shared chat.
- **Then** Each user can see the exchanged messages.

### E2E-PLAYWRIGHT-CHAT-APP-010

- **Source:** `suites/playwright-chat-app/test/e2e/chat-exchange.spec.ts`
- **Test:** `user B sees shared chat in their chat list`
- **Tags:** @svc_chat_app, @svc_gateway, @svc_organizations

**Scenario:** user B sees shared chat in their chat list

- **Given** A chat is shared with a second user.
- **When** User B opens their chat list.
- **Then** The shared chat appears in User B's list.

### E2E-PLAYWRIGHT-CHAT-APP-011

- **Source:** `suites/playwright-chat-app/test/e2e/chat-status-switch.spec.ts`
- **Test:** `switches from open to closed chat status`
- **Tags:** @svc_chat_app

**Scenario:** switches from open to closed chat status

- **Given** A chat is open.
- **When** The user changes the chat status to closed.
- **Then** The chat shows the closed status.

### E2E-PLAYWRIGHT-CHAT-APP-012

- **Source:** `suites/playwright-chat-app/test/e2e/chat-trace-link.spec.ts`
- **Test:** `view trace resolves tracing deep link (${scenario.name}) - codex`
- **Tags:** @svc_chat_app, @svc_tracing_app, @svc_agents_orchestrator, @svc_organizations

**Scenario:** view trace resolves tracing deep link (${scenario.name}) - codex

- **Given** A Codex-backed agent reply creates traceable activity in a chat.
- **When** The user opens the message trace link.
- **Then** The Tracing App opens the message deep link and either resolves to a finished run timeline or shows a stable resolving/no-run state.

### E2E-PLAYWRIGHT-CHAT-APP-013

- **Source:** `suites/playwright-chat-app/test/e2e/chat-trace-link.spec.ts`
- **Test:** `view trace resolves tracing deep link (${scenario.name}) - claude`
- **Tags:** @svc_chat_app, @svc_tracing_app, @svc_agents_orchestrator, @svc_organizations

**Scenario:** view trace resolves tracing deep link (${scenario.name}) - claude

- **Given** A Claude-backed agent reply creates traceable activity in a chat.
- **When** The user opens the message trace link.
- **Then** The Tracing App opens the message deep link and either resolves to a finished run timeline or shows a stable resolving/no-run state.

### E2E-PLAYWRIGHT-CHAT-APP-014

- **Source:** `suites/playwright-chat-app/test/e2e/chat-with-agent.spec.ts`
- **Test:** `chat with agent and receive reply`
- **Tags:** @svc_chat_app, @svc_agents_orchestrator

**Scenario:** chat with agent and receive reply

- **Given** A chat has an agent participant.
- **When** The user chats with the agent.
- **Then** The chat displays the agent reply.

### E2E-PLAYWRIGHT-CHAT-APP-015

- **Source:** `suites/playwright-chat-app/test/e2e/chats-list.spec.ts`
- **Test:** `renders chat list on load`
- **Tags:** @svc_chat_app

**Scenario:** renders chat list on load

- **Given** The user has access to chats.
- **When** The user opens the Chat App.
- **Then** The chat list renders on load.

### E2E-PLAYWRIGHT-CHAT-APP-016

- **Source:** `suites/playwright-chat-app/test/e2e/chats-list.spec.ts`
- **Test:** `participant picker shows available options`
- **Tags:** @svc_chat_app

**Scenario:** participant picker shows available options

- **Given** Participants are available for chat creation or filtering.
- **When** The user opens the participant picker.
- **Then** Available participant options are displayed.

### E2E-PLAYWRIGHT-CHAT-APP-017

- **Source:** `suites/playwright-chat-app/test/e2e/chats-list.spec.ts`
- **Test:** `redirects root to /chats`
- **Tags:** @svc_chat_app

**Scenario:** redirects root to /chats

- **Given** The user navigates to the Chat App root.
- **When** The app handles the root route.
- **Then** The browser is redirected to `/chats`.

### E2E-PLAYWRIGHT-CHAT-APP-018

- **Source:** `suites/playwright-chat-app/test/e2e/chats-list.spec.ts`
- **Test:** `navigates to chat detail`
- **Tags:** @svc_chat_app

**Scenario:** navigates to chat detail

- **Given** A chat is visible in the chat list.
- **When** The user selects the chat.
- **Then** The chat detail route and content are shown.

### E2E-PLAYWRIGHT-CHAT-APP-019

- **Source:** `suites/playwright-chat-app/test/e2e/file-upload.spec.ts`
- **Test:** `uploads a file and renders attachment`
- **Tags:** @svc_chat_app, @svc_files, @svc_media_proxy

**Scenario:** uploads a file and renders attachment

- **Given** A chat accepts file attachments.
- **When** The user uploads a file in the chat.
- **Then** The message renders the uploaded attachment.

### E2E-PLAYWRIGHT-CHAT-APP-020

- **Source:** `suites/playwright-chat-app/test/e2e/inline-media.spec.ts`
- **Test:** `renders mermaid diagrams inline`
- **Tags:** @svc_chat_app, @svc_media_proxy

**Scenario:** renders mermaid diagrams inline

- **Given** A chat message contains Mermaid media syntax.
- **When** The message is rendered.
- **Then** The Mermaid diagram appears inline.

### E2E-PLAYWRIGHT-CHAT-APP-021

- **Source:** `suites/playwright-chat-app/test/e2e/inline-media.spec.ts`
- **Test:** `renders vega-lite charts inline`
- **Tags:** @svc_chat_app, @svc_media_proxy

**Scenario:** renders vega-lite charts inline

- **Given** A chat message contains Vega-Lite media syntax.
- **When** The message is rendered.
- **Then** The Vega-Lite chart appears inline.

### E2E-PLAYWRIGHT-CHAT-APP-022

- **Source:** `suites/playwright-chat-app/test/e2e/inline-media.spec.ts`
- **Test:** `handles invalid mermaid input`
- **Tags:** @svc_chat_app, @svc_media_proxy

**Scenario:** handles invalid mermaid input

- **Given** A chat message contains invalid Mermaid input.
- **When** The message is rendered.
- **Then** The UI handles the invalid media without breaking the chat.

### E2E-PLAYWRIGHT-CHAT-APP-023

- **Source:** `suites/playwright-chat-app/test/e2e/inline-media.spec.ts`
- **Test:** `handles invalid vega-lite input`
- **Tags:** @svc_chat_app, @svc_media_proxy

**Scenario:** handles invalid vega-lite input

- **Given** A chat message contains invalid Vega-Lite input.
- **When** The message is rendered.
- **Then** The UI handles the invalid media without breaking the chat.

### E2E-PLAYWRIGHT-CHAT-APP-024

- **Source:** `suites/playwright-chat-app/test/e2e/inline-media.spec.ts`
- **Test:** `renders multiple inline media attachments`
- **Tags:** @svc_chat_app, @svc_media_proxy

**Scenario:** renders multiple inline media attachments

- **Given** A chat message contains multiple inline media attachments.
- **When** The message is rendered.
- **Then** All supported inline media attachments appear.

### E2E-PLAYWRIGHT-CHAT-APP-025

- **Source:** `suites/playwright-chat-app/test/e2e/organization-switching.spec.ts`
- **Test:** `switching organization updates chat list`
- **Tags:** @svc_chat_app, @svc_organizations

**Scenario:** switching organization updates chat list

- **Given** The user belongs to multiple organizations with different chats.
- **When** The user switches the selected organization.
- **Then** The chat list updates to the selected organization.

### E2E-PLAYWRIGHT-CHAT-APP-026

- **Source:** `suites/playwright-chat-app/test/e2e/sign-in.spec.ts`
- **Test:** `signs in via oidc redirect flow`
- **Tags:** @svc_chat_app

**Scenario:** signs in via oidc redirect flow

- **Given** The user is not signed in.
- **When** The user completes the OIDC redirect flow.
- **Then** The Chat App shows an authenticated session.

### E2E-PLAYWRIGHT-CHAT-APP-027

- **Source:** `suites/playwright-chat-app/test/e2e/sign-out.spec.ts`
- **Test:** `sign out clears oidc session storage`
- **Tags:** @svc_chat_app

**Scenario:** sign out clears oidc session storage

- **Given** The user is signed in.
- **When** The user signs out.
- **Then** OIDC session storage is cleared and the app is unauthenticated.
