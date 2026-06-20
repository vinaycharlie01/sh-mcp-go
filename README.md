<div align="center">

<img src="https://raw.githubusercontent.com/vinaycharlie01/sh-mcp-go/main/docs/assets/logo.svg" alt="sh-mcp-go logo" width="120" height="120" />

# sh-mcp-go

**AI-Native Kubernetes Deployment Orchestrator via Model Context Protocol**

[![Go Version](https://img.shields.io/badge/Go-1.25-00ADD8?style=for-the-badge&logo=go&logoColor=white)](https://go.dev)
[![MCP Protocol](https://img.shields.io/badge/MCP-Compatible-7C3AED?style=for-the-badge&logoColor=white)](https://modelcontextprotocol.io)
[![License](https://img.shields.io/badge/License-MIT-22C55E?style=for-the-badge)](LICENSE)
[![Docker](https://img.shields.io/badge/Image-UBI9%20Minimal-EE0000?style=for-the-badge&logo=redhat&logoColor=white)](https://ghcr.io/vinaycharlie01/sh-mcp-go)
[![Helm](https://img.shields.io/badge/Helm-SDK-0F1689?style=for-the-badge&logo=helm&logoColor=white)](https://helm.sh)
[![Kubernetes](https://img.shields.io/badge/Kubernetes-1.28%2B-326CE5?style=for-the-badge&logo=kubernetes&logoColor=white)](https://kubernetes.io)

> *Deploy to Kubernetes by talking to your AI. No scripts. No pipelines. Just conversation.*

[Quick Start](#-quick-start) · [Architecture](#-architecture) · [24 MCP Tools](#-mcp-tools) · [Configuration](#-configuration) · [Docker](#-container) · [Development](#-development)

</div>

---

## What is sh-mcp-go?

`sh-mcp-go` turns any [MCP-compatible AI client](https://modelcontextprotocol.io) — Claude Desktop, Cursor, Cline, or your own agent — into a full Kubernetes deployment operator.

The AI calls **24 purpose-built MCP tools** to install Helm charts, upgrade operators, roll back failed releases, run root-cause analysis, scan for security misconfigurations, and estimate resource requirements — all without leaving the chat window and without a single shell script.

```
You: "Deploy nginx-ingress to the prod cluster, wait for it to be healthy,
      then check if we need to upgrade cert-manager."

AI:  plan_deployment → install_chart → health_check → recommend_upgrade
     ✓ nginx-ingress 4.11.3 installed (3 replicas, healthy)
     ✓ cert-manager 1.14.2 → upgrade to 1.17.0 recommended (3 CVEs fixed)
```

---

## Features

| | |
|---|---|
| **24 MCP Tools** | Full deployment lifecycle: install, upgrade, rollback, uninstall, health-check, RCA, resource estimation, dependency analysis, security scan |
| **Pure Go** | No `kubectl` or `helm` binary required — uses the Helm SDK and `client-go` directly |
| **AI Planning** | `plan_deployment` converts natural language into an ordered, dependency-aware step graph |
| **Dual Transport** | stdio (local AI clients) and SSE (remote/web clients) |
| **Observability** | Prometheus metrics + OpenTelemetry tracing built in |
| **State Tracking** | SQLite-backed deployment history — lightweight, zero-dependency |
| **Multi-platform** | linux/amd64 and linux/arm64 container images on Red Hat UBI9 Minimal |

---

## Architecture

### System Overview

```mermaid
flowchart TB
    subgraph Clients["AI Clients"]
        A1[Claude Desktop]
        A2[Cursor / Cline]
        A3[Custom Agent]
    end

    subgraph Transport["MCP Transport"]
        T1[stdio]
        T2[SSE :8081]
    end

    subgraph MCP["MCP Adapter Layer"]
        M[24 Tool Handlers\ninternal/adapters/mcp]
    end

    subgraph App["Application Services"]
        D[Deployment Service\ninstall · upgrade · rollback]
        C[Cluster Service\ninventory · validate · scan]
        P[Planner Service\nplan_deployment]
    end

    subgraph Infra["Infrastructure Adapters"]
        H[Helm SDK Adapter\nno helm binary]
        K[Kubernetes client-go\nno kubectl binary]
        S[(SQLite\nState Store)]
    end

    subgraph Obs["Observability"]
        PR[Prometheus\n:8080/metrics]
        OT[OpenTelemetry\nTracing]
    end

    Clients --> Transport
    Transport --> MCP
    MCP --> App
    D --> H
    D --> K
    D --> S
    C --> K
    P --> D
    App --> Obs

    style Clients fill:#7C3AED,color:#fff
    style MCP fill:#0F1689,color:#fff
    style App fill:#065F46,color:#fff
    style Infra fill:#92400E,color:#fff
    style Obs fill:#1E3A5F,color:#fff
```

### Hexagonal Architecture

```mermaid
flowchart LR
    subgraph Domain["Domain Layer"]
        DE[Deployment\nAggregate]
        PL[Plan\nAggregate]
        CL[Cluster\nEntity]
    end

    subgraph Ports["Ports"]
        direction TB
        IP[Inbound Ports\nService Interfaces]
        OP[Outbound Ports\nHelmPort · K8sPort\nStoragePort]
    end

    subgraph InAdapters["Inbound Adapters"]
        MCP[MCP Server\nstdio / SSE]
    end

    subgraph OutAdapters["Outbound Adapters"]
        HA[Helm Adapter\nHelm SDK]
        KA[K8s Adapter\nclient-go]
        SA[SQLite Adapter\nmodernc/sqlite]
    end

    MCP -->|calls| IP
    IP --> Domain
    Domain --> OP
    OP -->|implemented by| HA
    OP -->|implemented by| KA
    OP -->|implemented by| SA

    style Domain fill:#065F46,color:#fff
    style Ports fill:#1E3A5F,color:#fff
    style InAdapters fill:#7C3AED,color:#fff
    style OutAdapters fill:#92400E,color:#fff
```

### Deployment Flow

```mermaid
sequenceDiagram
    participant AI as AI Client
    participant MCP as MCP Adapter
    participant SVC as Deployment Service
    participant HELM as Helm SDK
    participant K8S as Kubernetes API
    participant DB as SQLite

    AI->>MCP: install_chart(nginx-ingress, prod)
    MCP->>SVC: InstallChart(command)
    SVC->>DB: CreateDeployment(pending)
    SVC->>HELM: Install(chart, values)
    HELM->>K8S: Apply manifests
    K8S-->>HELM: Resources created
    HELM-->>SVC: Release info
    SVC->>DB: UpdateDeployment(success)
    SVC-->>MCP: DeploymentResult
    MCP-->>AI: ✓ nginx-ingress 4.11.3 installed
```

---

## Quick Start

### Prerequisites

| Requirement | Version |
|-------------|---------|
| Go | 1.25+ |
| Kubernetes cluster | 1.28+ |
| `~/.kube/config` | pointing at your cluster |

### Build and Run

```bash
# Clone
git clone https://github.com/vinaycharlie01/sh-mcp-go
cd sh-mcp-go

# Install mage (one-time)
go install github.com/magefile/mage@latest

# Build
mage build

# Run (stdio mode — for Claude Desktop)
./dist/sh-mcp-go
```

### Claude Desktop Integration

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

Restart Claude Desktop. The `sh-mcp-go` tool set will appear in the MCP tool panel.

See [`examples/claude-desktop.json`](examples/claude-desktop.json) for a full annotated example.

### Remote / SSE Mode

```yaml
# configs/local.yaml
mcp:
  transport: "sse"
  sse_addr: "0.0.0.0:8081"
```

```bash
./dist/sh-mcp-go --config configs/local.yaml
# MCP endpoint: http://localhost:8081/sse
```

---

## MCP Tools

### Tool Map

```mermaid
mindmap
  root((sh-mcp-go\n24 Tools))
    Chart Lifecycle
      install_chart
      upgrade_chart
      rollback_chart
      uninstall_chart
    Operator Lifecycle
      install_operator
      upgrade_operator
      rollback_operator
      delete_operator
    Planning
      plan_deployment
    Validation
      validate_cluster
      validate_release
    Inventory
      cluster_inventory
      release_inventory
    Analysis
      resource_estimation
      dependency_analysis
      security_scan
      health_check
      generate_rca
      analyze_failure
    Recommendations
      generate_values_yaml
      recommend_upgrade
      recommend_operator
    Status
      deployment_status
      release_status
```

### Tool Reference

| Category | Tool | Description |
|----------|------|-------------|
| **Chart Lifecycle** | `install_chart` | Install a Helm chart with full values support |
| | `upgrade_chart` | Upgrade an existing release |
| | `rollback_chart` | Roll back to a previous revision |
| | `uninstall_chart` | Remove a Helm release |
| **Operator Lifecycle** | `install_operator` | Install a Kubernetes operator |
| | `upgrade_operator` | Upgrade an operator release |
| | `rollback_operator` | Roll back an operator |
| | `delete_operator` | Delete an operator |
| **Planning** | `plan_deployment` | Convert natural language to ordered deployment step graph |
| **Validation** | `validate_cluster` | Run cluster prerequisite checks before deployment |
| | `validate_release` | Validate resource health for an existing release |
| **Inventory** | `cluster_inventory` | Full cluster inventory (nodes, namespaces, releases, CRDs) |
| | `release_inventory` | List all Helm releases with status |
| **Analysis** | `resource_estimation` | Estimate CPU/memory/storage for a chart |
| | `dependency_analysis` | Analyse chart dependency graph |
| | `security_scan` | Scan for RBAC issues, privilege escalation, and CVEs |
| | `health_check` | Check health of all resources in a release |
| | `generate_rca` | Generate root cause analysis for a failed deployment |
| | `analyze_failure` | Analyse failure with step-by-step remediation |
| **Recommendations** | `generate_values_yaml` | Generate a values.yaml skeleton for a chart |
| | `recommend_upgrade` | Advise whether and how to upgrade a release |
| | `recommend_operator` | Recommend an operator for a given workload type |
| **Status** | `deployment_status` | Tracked deployment status from state store |
| | `release_status` | Live Helm release status from the cluster |

Full parameter reference: [`docs/tools.md`](docs/tools.md)

---

## Container

The container image uses **Red Hat UBI9 Minimal** as its base — no Go toolchain inside, just the statically compiled binary. This keeps the image small and CVE-surface minimal.

```mermaid
flowchart LR
    A[mage buildLinux\ncross-compile\nlinux/amd64 + arm64] --> B[dist/linux_amd64/sh-mcp-go\ndist/linux_arm64/sh-mcp-go]
    B --> C[docker buildx\nARG TARGETARCH]
    C --> D[ubi9/ubi-minimal\n+ ca-certificates\n+ binary COPY only]
    D --> E[ghcr.io/vinaycharlie01/\nsh-mcp-go:latest]

    style A fill:#065F46,color:#fff
    style D fill:#EE0000,color:#fff
    style E fill:#1E3A5F,color:#fff
```

```bash
# Pull and run
docker pull ghcr.io/vinaycharlie01/sh-mcp-go:latest

docker run --rm \
  -v ~/.kube:/home/user/.kube:ro \
  -e KUBECONFIG=/home/user/.kube/config \
  ghcr.io/vinaycharlie01/sh-mcp-go:latest
```

---

## Configuration

All configuration lives in YAML with `SHMCP_` environment variable overrides.

```bash
cp configs/default.yaml configs/local.yaml
./dist/sh-mcp-go --config configs/local.yaml

# Or use environment variables
SHMCP_MCP_TRANSPORT=sse ./dist/sh-mcp-go
SHMCP_SERVER_PORT=9090 ./dist/sh-mcp-go
```

```yaml
# configs/default.yaml (key sections)
mcp:
  transport: "stdio"   # or "sse"
  sse_addr: "0.0.0.0:8081"

kubernetes:
  kubeconfig: ""       # defaults to ~/.kube/config
  context: ""          # defaults to current context

storage:
  sqlite:
    path: "./sh-mcp-go.db"

observability:
  metrics_addr: "0.0.0.0:8080"
  tracing_enabled: false
```

Full reference: [`docs/configuration.md`](docs/configuration.md)

---

## Development

### Mage Targets

```bash
go install github.com/magefile/mage@latest

mage build        # compile for current platform → dist/sh-mcp-go
mage buildLinux   # cross-compile linux/amd64 + linux/arm64 → dist/linux_*/
mage test         # run unit tests
mage lint         # golangci-lint
mage vet          # go vet
mage setup        # go mod download
mage clean        # remove dist/

mage docker:build # build multi-platform container image
mage docker:push  # push to GHCR
mage release      # goreleaser release
```

### Project Layout

```
sh-mcp-go/
├── cmd/sh-mcp-go/          entry point
├── configs/                 default configuration
├── docs/                    architecture, configuration, tool reference
├── examples/                Claude Desktop and tool invocation examples
├── internal/
│   ├── adapters/
│   │   ├── helm/            Helm SDK adapter (no helm binary)
│   │   ├── kubernetes/      client-go adapter (no kubectl binary)
│   │   ├── mcp/             MCP server + 24 tool handlers
│   │   ├── storage/sqlite/  SQLite repository
│   │   ├── events/          domain event publisher
│   │   └── observability/   Prometheus + OpenTelemetry
│   ├── application/
│   │   ├── deployment/      install/upgrade/rollback orchestration
│   │   ├── cluster/         cluster inspection service
│   │   └── planner/         AI deployment planner
│   ├── bootstrap/           dependency wiring
│   ├── domain/
│   │   ├── deployment/      Deployment aggregate + value objects
│   │   └── plan/            Plan aggregate + step entities
│   ├── infrastructure/
│   │   ├── config/          Viper-based config loader
│   │   ├── circuit/         Circuit breaker
│   │   ├── retry/           Retry policies
│   │   └── server/          HTTP metrics server
│   └── ports/
│       └── outbound/        HelmPort · K8sPort · StoragePort interfaces
└── pkg/
    ├── errors/              domain errors
    ├── logger/              slog-based structured logger
    └── version/             build version info
```

### CI Pipeline

```mermaid
flowchart LR
    PR[Pull Request] --> L[Lint\ngolangci-lint v2]
    PR --> T[Unit Tests]
    PR --> R[Race Tests]
    PR --> COV[Coverage]
    PR --> BEN[Benchmarks]
    PR --> SEC[Security\ngovulncheck + Trivy]
    PR --> SBOM[SBOM\nSyft]

    L --> B[Build\nmage build]
    T --> B

    B --> CTR[Container\nmage buildLinux\ndocker buildx\nUBI9 Minimal]

    style PR fill:#7C3AED,color:#fff
    style CTR fill:#EE0000,color:#fff
```

---

## License

MIT — see [LICENSE](LICENSE).

---

<div align="center">

Built with [Helm SDK](https://helm.sh) · [client-go](https://github.com/kubernetes/client-go) · [mcp-go](https://github.com/mark3labs/mcp-go) · [nava](https://github.com/nirantaraai/nava)

</div>
