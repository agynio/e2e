# agynio/e2e

End-to-end test suites for the Agyn platform.

## Microservice E2E status

| Service | E2E workflow |
|---------|--------------|
| [agent-state](https://github.com/agynio/agent-state) | [![E2E](https://github.com/agynio/agent-state/actions/workflows/e2e.yml/badge.svg?branch=main)](https://github.com/agynio/agent-state/actions/workflows/e2e.yml?query=branch:main) |
| [agents](https://github.com/agynio/agents) | [![E2E](https://github.com/agynio/agents/actions/workflows/e2e.yml/badge.svg?branch=main)](https://github.com/agynio/agents/actions/workflows/e2e.yml?query=branch:main) |
| [agents-orchestrator](https://github.com/agynio/agents-orchestrator) | [![E2E](https://github.com/agynio/agents-orchestrator/actions/workflows/e2e.yml/badge.svg?branch=main)](https://github.com/agynio/agents-orchestrator/actions/workflows/e2e.yml?query=branch:main) |
| [agn-cli](https://github.com/agynio/agn-cli) | [![E2E](https://github.com/agynio/agn-cli/actions/workflows/e2e.yml/badge.svg?branch=main)](https://github.com/agynio/agn-cli/actions/workflows/e2e.yml?query=branch:main) |
| [agynd-cli](https://github.com/agynio/agynd-cli) | [![E2E](https://github.com/agynio/agynd-cli/actions/workflows/e2e.yml/badge.svg?branch=main)](https://github.com/agynio/agynd-cli/actions/workflows/e2e.yml?query=branch:main) |
| [apps](https://github.com/agynio/apps) | [![E2E](https://github.com/agynio/apps/actions/workflows/e2e.yml/badge.svg?branch=main)](https://github.com/agynio/apps/actions/workflows/e2e.yml?query=branch:main) |
| [authorization](https://github.com/agynio/authorization) | [![E2E](https://github.com/agynio/authorization/actions/workflows/e2e.yml/badge.svg?branch=main)](https://github.com/agynio/authorization/actions/workflows/e2e.yml?query=branch:main) |
| [chat-app](https://github.com/agynio/chat-app) | [![E2E](https://github.com/agynio/chat-app/actions/workflows/e2e.yml/badge.svg?branch=main)](https://github.com/agynio/chat-app/actions/workflows/e2e.yml?query=branch:main) |
| [console-app](https://github.com/agynio/console-app) | [![E2E](https://github.com/agynio/console-app/actions/workflows/e2e.yml/badge.svg?branch=main)](https://github.com/agynio/console-app/actions/workflows/e2e.yml?query=branch:main) |
| [files](https://github.com/agynio/files) | [![E2E](https://github.com/agynio/files/actions/workflows/e2e.yml/badge.svg?branch=main)](https://github.com/agynio/files/actions/workflows/e2e.yml?query=branch:main) |
| [files-mcp](https://github.com/agynio/files-mcp) | [![E2E](https://github.com/agynio/files-mcp/actions/workflows/e2e.yml/badge.svg?branch=main)](https://github.com/agynio/files-mcp/actions/workflows/e2e.yml?query=branch:main) |
| [gateway](https://github.com/agynio/gateway) | [![E2E](https://github.com/agynio/gateway/actions/workflows/e2e.yml/badge.svg?branch=main)](https://github.com/agynio/gateway/actions/workflows/e2e.yml?query=branch:main) |
| [identity](https://github.com/agynio/identity) | [![E2E](https://github.com/agynio/identity/actions/workflows/e2e.yml/badge.svg?branch=main)](https://github.com/agynio/identity/actions/workflows/e2e.yml?query=branch:main) |
| [k8s-runner](https://github.com/agynio/k8s-runner) | [![E2E](https://github.com/agynio/k8s-runner/actions/workflows/e2e.yml/badge.svg?branch=main)](https://github.com/agynio/k8s-runner/actions/workflows/e2e.yml?query=branch:main) |
| [llm](https://github.com/agynio/llm) | [![E2E](https://github.com/agynio/llm/actions/workflows/e2e.yml/badge.svg?branch=main)](https://github.com/agynio/llm/actions/workflows/e2e.yml?query=branch:main) |
| [llm-proxy](https://github.com/agynio/llm-proxy) | [![E2E](https://github.com/agynio/llm-proxy/actions/workflows/e2e.yml/badge.svg?branch=main)](https://github.com/agynio/llm-proxy/actions/workflows/e2e.yml?query=branch:main) |
| [media-proxy](https://github.com/agynio/media-proxy) | [![E2E](https://github.com/agynio/media-proxy/actions/workflows/e2e.yml/badge.svg?branch=main)](https://github.com/agynio/media-proxy/actions/workflows/e2e.yml?query=branch:main) |
| [metering](https://github.com/agynio/metering) | [![E2E](https://github.com/agynio/metering/actions/workflows/e2e.yml/badge.svg?branch=main)](https://github.com/agynio/metering/actions/workflows/e2e.yml?query=branch:main) |
| [organizations](https://github.com/agynio/organizations) | [![E2E](https://github.com/agynio/organizations/actions/workflows/e2e.yml/badge.svg?branch=main)](https://github.com/agynio/organizations/actions/workflows/e2e.yml?query=branch:main) |
| [runners](https://github.com/agynio/runners) | [![E2E](https://github.com/agynio/runners/actions/workflows/e2e.yml/badge.svg?branch=main)](https://github.com/agynio/runners/actions/workflows/e2e.yml?query=branch:main) |
| [tracing-app](https://github.com/agynio/tracing-app) | [![E2E](https://github.com/agynio/tracing-app/actions/workflows/e2e.yml/badge.svg?branch=main)](https://github.com/agynio/tracing-app/actions/workflows/e2e.yml?query=branch:main) |

## Playwright suites

### tracing-app

Located in `suites/playwright-tracing-app` with tags `svc_tracing_app`, `svc_agents_orchestrator`, and `smoke`
(default when `TAGS` is empty). Provide a tracing-app URL that serves `/env.js` via `E2E_BASE_URL`
(defaults to `https://tracing.${E2E_DOMAIN:-agyn.dev}` in-cluster).
Optional OIDC overrides:

- `E2E_OIDC_AUTHORITY`
- `E2E_OIDC_CLIENT_ID`
- `E2E_OIDC_SCOPE`

Optional domain override:

- `E2E_DOMAIN` (falls back to `DOMAIN`, then `agyn.dev`)
- `E2E_INGRESS_PORT` (falls back to `INGRESS_PORT`, then `PORT`, then `2496`; used for runner-reachable ingress URLs such as `LLM_PROXY_URL`)
- `LLM_PROXY_URL` (default `https://llm.${E2E_DOMAIN:-agyn.dev}:${E2E_INGRESS_PORT:-2496}`)

Full-chain tests use `AGN_INIT_IMAGE` (default `ghcr.io/agynio/agent-init-agn:latest`) for agn,
`CODEX_INIT_IMAGE` (default `ghcr.io/agynio/agent-init-codex:latest`) for codex,
`CLAUDE_INIT_IMAGE` (default `ghcr.io/agynio/agent-init-claude:latest`) for claude,
and `AGN_EXPOSE_INIT_IMAGE` (default `ghcr.io/agynio/agent-init-agn:latest`) for go-core expose.
For exact reproducibility, set `*_INIT_IMAGE` to a pinned patch tag
(for example, `ghcr.io/agynio/agent-init-agn:0.4.15`) or an image digest
(for example, `ghcr.io/agynio/agent-init-agn@sha256:<digest>`).

Example runs:

```bash
TAGS=smoke devspace run test-e2e
TAGS=svc_tracing_app devspace run test-e2e
TAGS=svc_agents_orchestrator devspace run test-e2e
TAGS=svc_llm devspace run test-e2e
TAGS=svc_llm_proxy devspace run test-e2e
```
