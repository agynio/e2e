# E2E Glossary

| Term | Meaning in E2E docs | Architecture/product mapping |
| --- | --- | --- |
| Agent | Runtime participant that receives messages, may call tools or LLMs, and returns responses. | Agents / agents-orchestrator product area. |
| Agent init image | Container image that prepares an agent runtime for a specific protocol or CLI. | Runtime bootstrap and workload startup. |
| API token | User-created token accepted by the gateway for API access. | Identity and gateway authentication. |
| Chat | User-facing conversation in Chat App, usually backed by gateway thread/message APIs. | Product chat experience and thread model. |
| ConnectRPC | RPC transport used by generated clients and gateway APIs. | API boundary contract. |
| Console App | Administrative web UI for organizations, agents, runners, users, usage, and settings. | Console product surface. |
| Files service | Service that stores uploaded file metadata and content. | File storage capability. |
| Gateway | Public API boundary for authenticated platform requests. | Gateway architecture boundary. |
| Image pull secret | Registry credential attached to workloads, agents, MCPs, hooks, or Terraform-managed resources. | Workload/container registry integration. |
| LLM proxy | Service that routes model calls to configured LLM endpoints. | LLM product and proxy service. |
| MCP | Model Context Protocol tool server configuration attached to an agent. | Tool integration capability. |
| Media proxy | Service that safely proxies uploaded or external media for browser display. | Media delivery capability. |
| Metering | Usage recording and query surface for product and platform usage. | Metering service and usage dashboards. |
| Organization | Tenant boundary used to scope users, agents, chats, threads, runners, and resources. | Organization/product tenancy model. |
| Runner | Execution backend that starts, inspects, streams, and stops workloads. | Runners and k8s-runner services. |
| Terraform provider | Infrastructure-as-code provider for managing Agyn resources. | Provider/product integration. |
| Thread | Conversation record used by agent orchestration and surfaced in product UIs. | Threads/message domain model. |
| Trace / run | Observability record that links messages, LLM calls, tools, and execution events. | Tracing product surface. |
| Ziti expose | Network exposure path used to make agent-hosted services reachable. | Secure service exposure capability. |
