# sh-mcp-go

**AI-Native Kubernetes Deployment Orchestrator via Model Context Protocol**

`sh-mcp-go` exposes a Kubernetes/Helm deployment engine as a set of MCP tools that any MCP-compatible AI client (Claude Desktop, Cursor, Cline, etc.) can call directly. The AI composes those tools to plan, install, upgrade, roll back, and analyse Helm-based workloads on any Kubernetes cluster — all without leaving the chat window.

---

## Features

- **24 MCP tools** covering the full deployment lifecycle: install, upgrade, rollback, uninstall, health-check, RCA, resource estimation, dependency analysis, security scan, and more
- **Pure-Go implementation** — no `kubectl` or `helm` binary required; uses the Helm SDK and `client-go` directly
- **AI-driven planning** — `plan_deployment` turns natural language into an ordered, dependency-aware step graph
- **Dual transport** — stdio (for local AI clients) and SSE (for remote clients)
- **Observability** — Prometheus metrics and OpenTelemetry tracing built in
- **SQLite state store** — lightweight, file-based deployment tracking

---

## Quick Start

### Prerequisites

| Tool | Version |
|------|---------|
| Go | 1.25+ |
| Kubernetes cluster | 1.28+ |
| `~/.kube/config` | pointing at your cluster |

### Build

```bash
git clone https://github.com/vinaycharlie01/sh-mcp-go
cd sh-mcp-go
go build -o sh-mcp-go ./cmd/server
```

### Run (stdio — for Claude Desktop)

```bash
./sh-mcp-go
```

The binary reads `./configs/default.yaml` (or `/etc/sh-mcp-go/config.yaml`) and starts the MCP server on stdin/stdout.

### Run (SSE — for remote clients)

```yaml
# configs/default.yaml
mcp:
  transport: "sse"
  sse_addr: "0.0.0.0:8081"
```

```bash
./sh-mcp-go
# MCP endpoint: http://localhost:8081/sse
```

---

## Claude Desktop Integration

Add to `~/Library/Application Support/Claude/claude_desktop_config.json` (macOS) or `%APPDATA%\Claude\claude_desktop_config.json` (Windows):

```json
{
  "mcpServers": {
    "sh-mcp-go": {
      "command": "/usr/local/bin/sh-mcp-go",
      "args": [],
      "env": {
        "KUBECONFIG": "/home/user/.kube/config"
      }
    }
  }
}
```

Restart Claude Desktop. You will see "sh-mcp-go" listed as an active MCP server in the tool panel.

See [`examples/claude-desktop.json`](examples/claude-desktop.json) for a full annotated example.

---

## Architecture

```
┌─────────────────────────────────────────────────────┐
│  AI Client  (Claude Desktop / Cursor / Cline)        │
└────────────────────┬────────────────────────────────┘
                     │  MCP (stdio or SSE)
┌────────────────────▼────────────────────────────────┐
│  MCP Adapter  (internal/adapters/mcp)                │
│  24 tool handlers → application services             │
└──────┬──────────────────────┬───────────────────────┘
       │                      │
┌──────▼──────┐        ┌──────▼──────┐
│ Deployment  │        │  Cluster    │
│  Service    │        │  Service    │
│ (app layer) │        │ (app layer) │
└──────┬──────┘        └──────┬──────┘
       │                      │
┌──────▼──────┐        ┌──────▼──────┐
│ Helm Adapter│        │  K8s Adapter│
│ (Helm SDK)  │        │ (client-go) │
└─────────────┘        └─────────────┘
       │
┌──────▼──────┐
│  SQLite     │
│  Repository │
└─────────────┘
```

The project follows **Hexagonal Architecture** (Ports & Adapters) with Domain-Driven Design aggregates. See [`docs/architecture.md`](docs/architecture.md) for a detailed explanation.

---

## MCP Tools Overview

| Category | Tool | Description |
|----------|------|-------------|
| Chart lifecycle | `install_chart` | Install a Helm chart |
| | `upgrade_chart` | Upgrade a release |
| | `rollback_chart` | Roll back to a previous revision |
| | `uninstall_chart` | Remove a release |
| Operator lifecycle | `install_operator` | Install a Kubernetes operator |
| | `upgrade_operator` | Upgrade an operator |
| | `rollback_operator` | Roll back an operator |
| | `delete_operator` | Delete an operator |
| Planning | `plan_deployment` | Generate AI deployment plan from natural language |
| Validation | `validate_cluster` | Run cluster prerequisite checks |
| | `validate_release` | Validate release resource health |
| Inventory | `cluster_inventory` | Full cluster inventory (nodes, namespaces, releases, CRDs) |
| | `release_inventory` | List Helm releases |
| Analysis | `resource_estimation` | Estimate CPU/memory/storage for a chart |
| | `dependency_analysis` | Analyse chart dependencies |
| | `security_scan` | Scan for RBAC/privilege/CVE issues |
| | `health_check` | Check resource health for a release |
| | `generate_rca` | Generate root cause analysis |
| | `analyze_failure` | Analyse deployment failure with remediation steps |
| Recommendations | `generate_values_yaml` | Generate values.yaml skeleton |
| | `recommend_upgrade` | Recommend whether/how to upgrade |
| | `recommend_operator` | Recommend an operator for a workload type |
| Status | `deployment_status` | Get tracked deployment status |
| | `release_status` | Get Helm release status from cluster |

Full parameter reference: [`docs/tools.md`](docs/tools.md)

---

## Configuration

All configuration is in YAML. Copy `configs/default.yaml` and adjust:

```bash
cp configs/default.yaml configs/local.yaml
./sh-mcp-go --config configs/local.yaml
```

Environment variable overrides use the prefix `SHMCP_` with `.` replaced by `_`:

```bash
SHMCP_MCP_TRANSPORT=sse ./sh-mcp-go
SHMCP_SERVER_PORT=9090 ./sh-mcp-go
```

Full reference: [`docs/configuration.md`](docs/configuration.md)

---

## Development

```bash
# Run tests
go test ./...

# Lint
golangci-lint run

# Build
go build -o sh-mcp-go ./cmd/server
```

### Project Layout

```
cmd/server/          entry point
configs/             default configuration
docs/                architecture and reference docs
examples/            Claude Desktop and tool invocation examples
internal/
  adapters/
    helm/            Helm SDK adapter
    kubernetes/      client-go adapter
    mcp/             MCP server and tool handlers
    storage/sqlite/  SQLite repository
    events/          event publisher
    observability/   metrics and tracing
  application/
    deployment/      install/upgrade/rollback service
    cluster/         cluster inspection service
    planner/         AI deployment planner
  bootstrap/         dependency wiring
  domain/
    deployment/      Deployment aggregate and value objects
    plan/            Plan aggregate and step entities
  infrastructure/
    config/          Viper-based config loader
    retry/           Retry policies
    server/          HTTP metrics server
  ports/
    inbound/         Service interfaces (inbound)
    outbound/        Port interfaces (Helm, K8s)
pkg/
  logger/            slog-based structured logger
```

---

## License

MIT
