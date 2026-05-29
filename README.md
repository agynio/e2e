# agynio/e2e

End-to-end test suites for the Agyn platform.

This repository contains the shared E2E runner plus suite definitions for
service, UI, CLI, and Terraform provider coverage. The runner discovers
`suites/*/suite.yaml`, selects suites from `TAGS`, and executes each selected
suite in a Kubernetes pod.

For BDD coverage, tag conventions, and traceability references, see the
[E2E BDD documentation](docs/e2e/README.md).

## Run

Run the selected suites through DevSpace:

```sh
devspace run test-e2e
```

Run suites for one or more tags:

```sh
devspace run test-e2e --tag smoke
devspace run test-e2e --tag svc_gateway --tag svc_agents_orchestrator
```

Useful environment variables:

- `E2E_NAMESPACE`: Kubernetes namespace used for runner pods. Defaults to
  `platform`.
- `E2E_SUITES`: optional space-separated suite names to limit discovery, for
  example `playwright go-core`.
- `TAGS`: space-separated tags used by suite selection. Prefer `--tag` when
  invoking DevSpace locally.
- `E2E_DOMAIN`: app domain used by Playwright suite defaults. Defaults to
  `agyn.dev`.
- `E2E_INGRESS_PORT`: ingress port used by external service defaults. Falls
  back to `INGRESS_PORT`, then `PORT`, then `2496`.

## Suite selection

Each suite declares a `select` script in `suite.yaml`.

- With an empty `TAGS`, only suites that explicitly emit a default selector run.
- With `TAGS` set, a suite runs when any requested tag matches one of its suite
  tags.
- If `TAGS` is non-empty and no suite matches, the runner fails instead of
  silently doing nothing.
- Playwright suites convert selected tags to Playwright `--grep` expressions
  such as `@smoke|@svc_gateway`.
- Go suites convert selected tags to Go build tags alongside the shared `e2e`
  build tag.

## Suites

<!-- markdownlint-disable MD013 -->

| Suite | Default `TAGS` behavior | Suite tags | Required env vars | Playwright base URL default |
| --- | --- | --- | --- | --- |
| `go-agn-cli` | Not selected by default. Runs when `TAGS` includes `svc_agn_cli`. | `svc_agn_cli` | None in `suite.yaml`. Runtime expects `AGN_BINARY`, or `./bin/agn` by default. | N/A |
| `go-core` | Selected as `smoke`; runs Go tests with `e2e smoke`. | `svc_agents_orchestrator`, `svc_runners`, `svc_metering`, `svc_k8s_runner`, `svc_organizations`, `svc_files`, `svc_gateway`, `svc_media_proxy`, `svc_llm`, `svc_llm_proxy`, `smoke` | `AGN_INIT_IMAGE`, `CODEX_INIT_IMAGE`, `CLAUDE_INIT_IMAGE`, `AGN_EXPOSE_INIT_IMAGE` | N/A |
| `go-terraform` | Selected as `svc_gateway`; runs Go tests with `e2e svc_gateway tf_provider_agyn` when `TAGS` is empty. | `svc_gateway`, `tf_provider_agyn` | None in `suite.yaml`. | N/A |
| `playwright` | Selected by default and runs the full Playwright suite with no grep. | `svc_console`, `svc_gateway`, `svc_threads`, `svc_metering`, `svc_identity`, `smoke` | None in `suite.yaml`. | `https://console.${E2E_DOMAIN:-agyn.dev}` |
| `playwright-chat-app` | Not selected by default. Runs when `TAGS` matches one of its suite tags. | `svc_chat_app`, `svc_gateway`, `svc_agents_orchestrator`, `svc_organizations`, `svc_files`, `svc_media_proxy`, `svc_tracing_app` | `CODEX_INIT_IMAGE` | `https://chat.${E2E_DOMAIN:-agyn.dev}` |
| `playwright-tracing-app` | Selected as `smoke`; runs Playwright with `--grep @smoke`. | `svc_tracing_app`, `svc_agents_orchestrator`, `smoke` | `AGN_INIT_IMAGE`, `CODEX_INIT_IMAGE`, `CLAUDE_INIT_IMAGE` | `https://tracing.${E2E_DOMAIN:-agyn.dev}` |

<!-- markdownlint-enable MD013 -->

## Microservice E2E status

| Service | E2E | Service | E2E | Service | E2E |
| --- | --- | --- | --- | --- | --- |
| [agent-state](https://github.com/agynio/agent-state) | [![E2E](https://github.com/agynio/agent-state/actions/workflows/e2e.yml/badge.svg?branch=main)](https://github.com/agynio/agent-state/actions/workflows/e2e.yml?query=branch:main) | [agents](https://github.com/agynio/agents) | [![E2E](https://github.com/agynio/agents/actions/workflows/e2e.yml/badge.svg?branch=main)](https://github.com/agynio/agents/actions/workflows/e2e.yml?query=branch:main) | [agents-orchestrator](https://github.com/agynio/agents-orchestrator) | [![E2E](https://github.com/agynio/agents-orchestrator/actions/workflows/e2e.yml/badge.svg?branch=main)](https://github.com/agynio/agents-orchestrator/actions/workflows/e2e.yml?query=branch:main) |
| [agn-cli](https://github.com/agynio/agn-cli) | [![E2E](https://github.com/agynio/agn-cli/actions/workflows/e2e.yml/badge.svg?branch=main)](https://github.com/agynio/agn-cli/actions/workflows/e2e.yml?query=branch:main) | [agynd-cli](https://github.com/agynio/agynd-cli) | [![E2E](https://github.com/agynio/agynd-cli/actions/workflows/e2e.yml/badge.svg?branch=main)](https://github.com/agynio/agynd-cli/actions/workflows/e2e.yml?query=branch:main) | [apps](https://github.com/agynio/apps) | [![E2E](https://github.com/agynio/apps/actions/workflows/e2e.yml/badge.svg?branch=main)](https://github.com/agynio/apps/actions/workflows/e2e.yml?query=branch:main) |
| [authorization](https://github.com/agynio/authorization) | [![E2E](https://github.com/agynio/authorization/actions/workflows/e2e.yml/badge.svg?branch=main)](https://github.com/agynio/authorization/actions/workflows/e2e.yml?query=branch:main) | [chat-app](https://github.com/agynio/chat-app) | [![E2E](https://github.com/agynio/chat-app/actions/workflows/e2e.yml/badge.svg?branch=main)](https://github.com/agynio/chat-app/actions/workflows/e2e.yml?query=branch:main) | [console-app](https://github.com/agynio/console-app) | [![E2E](https://github.com/agynio/console-app/actions/workflows/e2e.yml/badge.svg?branch=main)](https://github.com/agynio/console-app/actions/workflows/e2e.yml?query=branch:main) |
| [files](https://github.com/agynio/files) | [![E2E](https://github.com/agynio/files/actions/workflows/e2e.yml/badge.svg?branch=main)](https://github.com/agynio/files/actions/workflows/e2e.yml?query=branch:main) | [files-mcp](https://github.com/agynio/files-mcp) | [![E2E](https://github.com/agynio/files-mcp/actions/workflows/e2e.yml/badge.svg?branch=main)](https://github.com/agynio/files-mcp/actions/workflows/e2e.yml?query=branch:main) | [gateway](https://github.com/agynio/gateway) | [![E2E](https://github.com/agynio/gateway/actions/workflows/e2e.yml/badge.svg?branch=main)](https://github.com/agynio/gateway/actions/workflows/e2e.yml?query=branch:main) |
| [identity](https://github.com/agynio/identity) | [![E2E](https://github.com/agynio/identity/actions/workflows/e2e.yml/badge.svg?branch=main)](https://github.com/agynio/identity/actions/workflows/e2e.yml?query=branch:main) | [k8s-runner](https://github.com/agynio/k8s-runner) | [![E2E](https://github.com/agynio/k8s-runner/actions/workflows/e2e.yml/badge.svg?branch=main)](https://github.com/agynio/k8s-runner/actions/workflows/e2e.yml?query=branch:main) | [llm](https://github.com/agynio/llm) | [![E2E](https://github.com/agynio/llm/actions/workflows/e2e.yml/badge.svg?branch=main)](https://github.com/agynio/llm/actions/workflows/e2e.yml?query=branch:main) |
| [llm-proxy](https://github.com/agynio/llm-proxy) | [![E2E](https://github.com/agynio/llm-proxy/actions/workflows/e2e.yml/badge.svg?branch=main)](https://github.com/agynio/llm-proxy/actions/workflows/e2e.yml?query=branch:main) | [media-proxy](https://github.com/agynio/media-proxy) | [![E2E](https://github.com/agynio/media-proxy/actions/workflows/e2e.yml/badge.svg?branch=main)](https://github.com/agynio/media-proxy/actions/workflows/e2e.yml?query=branch:main) | [metering](https://github.com/agynio/metering) | [![E2E](https://github.com/agynio/metering/actions/workflows/e2e.yml/badge.svg?branch=main)](https://github.com/agynio/metering/actions/workflows/e2e.yml?query=branch:main) |
| [organizations](https://github.com/agynio/organizations) | [![E2E](https://github.com/agynio/organizations/actions/workflows/e2e.yml/badge.svg?branch=main)](https://github.com/agynio/organizations/actions/workflows/e2e.yml?query=branch:main) | [runners](https://github.com/agynio/runners) | [![E2E](https://github.com/agynio/runners/actions/workflows/e2e.yml/badge.svg?branch=main)](https://github.com/agynio/runners/actions/workflows/e2e.yml?query=branch:main) | [tracing-app](https://github.com/agynio/tracing-app) | [![E2E](https://github.com/agynio/tracing-app/actions/workflows/e2e.yml/badge.svg?branch=main)](https://github.com/agynio/tracing-app/actions/workflows/e2e.yml?query=branch:main) |
