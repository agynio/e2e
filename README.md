# agynio/e2e

End-to-end test suites for the Agyn platform.

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

- `E2E_DOMAIN`

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
```
