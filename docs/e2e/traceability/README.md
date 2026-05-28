# E2E Traceability

Traceability files map architecture/product anchors and service tags to documented BDD cases. The anchors are intentionally stable labels until deeper links into `agynio/architecture` are available.

- [Service coverage](service-coverage.md)
- [Architecture and product anchors](architecture-product-anchors.md)

## Suite summary

| Suite | Cases | Tags |
| --- | ---: | --- |
| [Go AGN CLI](../suites/go-agn-cli.md) | 6 | @svc_agn_cli |
| [Go Core](../suites/go-core.md) | 89 | @svc_agents_orchestrator, @svc_runners, @svc_metering, @svc_k8s_runner, @svc_organizations, @svc_files, @svc_gateway, @svc_media_proxy, @svc_llm, @svc_llm_proxy, @smoke |
| [Go Terraform Provider](../suites/go-terraform.md) | 47 | @svc_gateway, @tf_provider_agyn |
| [Playwright Console App](../suites/playwright.md) | 41 | @svc_console, @svc_gateway, @svc_threads, @svc_metering, @svc_identity, @smoke |
| [Playwright Chat App](../suites/playwright-chat-app.md) | 27 | @svc_chat_app, @svc_gateway, @svc_agents_orchestrator, @svc_organizations, @svc_files, @svc_media_proxy, @svc_tracing_app |
| [Playwright Tracing App](../suites/playwright-tracing-app.md) | 8 | @svc_tracing_app, @svc_agents_orchestrator, @smoke |
