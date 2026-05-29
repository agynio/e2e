# E2E BDD Documentation

This directory documents what the E2E suites validate in behavior-driven (BDD) language. It is intended for humans and agents comparing test coverage against architecture and product requirements; it does not document implementation mechanics.

## Navigation

| Suite ID | Suite doc | Source suite |
| --- | --- | --- |
| E2E-SUITE-GO-AGN-CLI | [Go AGN CLI](suites/go-agn-cli.md) | `suites/go-agn-cli/suite.yaml` |
| E2E-SUITE-GO-CORE | [Go Core](suites/go-core.md) | `suites/go-core/suite.yaml` |
| E2E-SUITE-GO-TERRAFORM | [Go Terraform Provider](suites/go-terraform.md) | `suites/go-terraform/suite.yaml` |
| E2E-SUITE-PLAYWRIGHT | [Playwright Console App](suites/playwright.md) | `suites/playwright/suite.yaml` |
| E2E-SUITE-PLAYWRIGHT-CHAT-APP | [Playwright Chat App](suites/playwright-chat-app.md) | `suites/playwright-chat-app/suite.yaml` |
| E2E-SUITE-PLAYWRIGHT-TRACING-APP | [Playwright Tracing App](suites/playwright-tracing-app.md) | `suites/playwright-tracing-app/suite.yaml` |

Additional references:

- [Glossary](glossary.md)
- [Traceability index](traceability/README.md)
- [Service coverage](traceability/service-coverage.md)
- [Architecture and product anchors](traceability/architecture-product-anchors.md)

## DEV/E2E-only diagnostics resources

The `ziti-management-diagnostics` identity and Kubernetes secret are test-only
diagnostics resources. They are allowed only in development and E2E bootstrap
deployments, must be guarded by the bootstrap Terraform variable
`enable_ziti_management_diagnostics` defaulting to `false`, and must not exist
in production deployments.

## Tag glossary

Tags are selected through suite-level `TAGS` filtering. Suite docs list suite tags from `suites/*/suite.yaml`; individual case tags mirror the nearest Playwright `test.describe` tag or the service area implied by the Go test file.

| Tag family | Meaning |
| --- | --- |
| `@svc_*` | Service-oriented coverage tag. The suffix should match the platform service name used by suite selection, for example `@svc_gateway`, `@svc_agents_orchestrator`, or `@svc_chat_app`. |
| `@smoke` | Minimal high-signal coverage expected to run by default for smoke-capable suites. |
| `@tf_provider_agyn` | Terraform provider acceptance coverage for the Agyn provider. |
| `@issue*` | Regression coverage tied to a GitHub issue or historical bug. |

## Stable ID rules

- Suite IDs use `E2E-SUITE-<SUITE-NAME>` with the suite directory name uppercased and hyphenated.
- Case IDs use `E2E-<SUITE-NAME>-NNN` and are stable within a suite doc.
- Add new cases at the end of the relevant suite unless a deliberate renumbering migration is documented.
- Each Playwright `test(...)` case and each Go E2E top-level test or documented subtest from the suite runner's E2E test paths appears exactly once.
- Internal helper/unit tests outside the E2E suite run, such as `internal/` packages, are excluded from this inventory.
- Parameterized tests are expanded when the parameter value is part of the observable behavior, for example Codex and Claude trace-link scenarios.

## BDD documentation rules

- Describe externally observable behavior using Given/When/Then.
- Avoid internal implementation details unless they are the observable contract being tested, such as serialized API payload shape.
- Keep uncertain descriptions in the inventory and mark the affected sentence with `TODO: clarify`.
- Link suite docs back to source test files so implementation details can be inspected separately.
