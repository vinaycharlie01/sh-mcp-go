# Architecture

## Overview

`sh-mcp-go` is structured around **Hexagonal Architecture** (also called Ports and Adapters) combined with **Domain-Driven Design** (DDD) aggregates. The core idea: business logic lives in the centre and knows nothing about the outside world. External systems (Kubernetes, Helm, SQLite, MCP clients) talk to the core through explicit interfaces called *ports*.

```
                        ┌─────────────────────────────┐
                        │         DOMAIN               │
                        │  deployment.Deployment       │
                        │  plan.Plan / plan.Step       │
                        │  (pure Go, no dependencies)  │
                        └──────────────┬───────────────┘
                                       │
                 ┌─────────────────────▼──────────────────────┐
                 │           APPLICATION SERVICES              │
                 │  deployment.Service  cluster.Service        │
                 │  planner.Service                            │
                 │  (orchestrates domain + calls ports)        │
                 └──┬────────────┬──────────────┬─────────────┘
                    │            │              │
           ┌────────▼──┐  ┌──────▼────┐  ┌─────▼───────┐
           │ HelmPort  │  │  K8sPort  │  │  Repository │
           │ (outbound)│  │ (outbound)│  │  (outbound) │
           └────────┬──┘  └──────┬────┘  └─────┬───────┘
                    │            │              │
           ┌────────▼──┐  ┌──────▼────┐  ┌─────▼───────┐
           │  Helm     │  │  K8s      │  │  SQLite     │
           │  Adapter  │  │  Adapter  │  │  Adapter    │
           └───────────┘  └───────────┘  └─────────────┘

  ┌─────────────────────────────────────┐
  │           MCP Adapter               │
  │  (inbound — implements tool calls)  │
  │  → calls application services       │
  └─────────────────────────────────────┘
```

---

## Layers

### Domain (`internal/domain/`)

The domain layer holds the business concepts with no external imports.

**`deployment` package**

- `Deployment` — the aggregate root. Tracks a Helm release lifecycle: `PENDING → DEPLOYING → SUCCEEDED / FAILED / ROLLED_BACK`.
- `ReleaseName`, `Namespace`, `ChartReference`, `Values` — value objects with validation.
- `ID` — typed identifier to prevent mixing with other string IDs.

**`plan` package**

- `Plan` — aggregate root for an AI-generated deployment plan.
- `Step` — a single atomic action (`INSTALL_CHART`, `CREATE_NAMESPACE`, etc.) with its own status and dependency list.
- `ReadySteps()` — returns steps whose dependencies are all `SUCCEEDED`/`SKIPPED`, enabling parallel execution.
- Status machine: `DRAFT → APPROVED → EXECUTING → COMPLETED / FAILED / ABORTED`.

### Application (`internal/application/`)

Thin orchestration layer. Services coordinate domain aggregates and ports but contain no infrastructure details.

| Service | Responsibility |
|---------|----------------|
| `deployment.Service` | Install, upgrade, rollback, uninstall Helm releases; persists `Deployment` aggregate |
| `cluster.Service` | Cluster inventory, health summary, resource estimation, RCA |
| `planner.Service` | Translates natural language intent into a `Plan` with ordered `Step`s |

### Ports (`internal/ports/`)

Go interfaces that the application layer depends on. Adapters implement them.

**Outbound ports** (application → infrastructure):

```go
// outbound.HelmPort
type HelmPort interface {
    InstallChart(ctx, cmd) (*release.Release, error)
    UpgradeChart(ctx, cmd) (*release.Release, error)
    RollbackChart(ctx, cmd) error
    UninstallChart(ctx, cmd) error
    ListReleases(ctx, namespace) ([]*release.Release, error)
    GetRelease(ctx, name, namespace) (*release.Release, error)
    ResolveVersion(ctx, chart, repo, constraint) (string, error)
    GenerateValues(ctx, chart, repo, version) (map[string]any, error)
}

// outbound.K8sPort
type K8sPort interface {
    GetNodes(ctx) ([]NodeInfo, error)
    GetNamespaces(ctx) ([]string, error)
    GetPodLogs(ctx, namespace, podName) (string, error)
    GetEvents(ctx, namespace) ([]EventInfo, error)
    ListCRDs(ctx) ([]string, error)
}

// outbound.Repository
type Repository interface {
    Save(ctx, *deployment.Deployment) error
    FindByID(ctx, deployment.ID) (*deployment.Deployment, error)
    FindByReleaseName(ctx, ReleaseName, Namespace) (*deployment.Deployment, error)
    ListByNamespace(ctx, Namespace) ([]*deployment.Deployment, error)
    Delete(ctx, deployment.ID) error
}
```

### Adapters (`internal/adapters/`)

Implementations of the ports.

| Adapter | Port | Technology |
|---------|------|------------|
| `helm/` | `HelmPort` | `helm.sh/helm/v3` SDK (pure Go — no helm binary) |
| `kubernetes/` | `K8sPort` | `k8s.io/client-go` (pure Go — no kubectl binary) |
| `storage/sqlite/` | `Repository` | `modernc.org/sqlite` (pure Go — no CGo) |
| `mcp/` | — | `github.com/mark3labs/mcp-go` MCP server |
| `events/` | `EventPublisher` | structured slog logging |
| `observability/` | — | Prometheus + OpenTelemetry |

### Infrastructure (`internal/infrastructure/`)

Cross-cutting concerns with no business logic.

- `config/` — Viper-based YAML loader with env override and hot-reload (`Watch`)
- `retry/` — configurable retry policies (`DefaultHelmPolicy`, `DefaultK8sPolicy`) using `avast/retry-go`
- `server/` — HTTP server for the Prometheus `/metrics` endpoint

### Bootstrap (`internal/bootstrap/`)

Manual dependency injection wiring. `Build(ctx, cfg)` constructs every component in the correct order and returns an `App` struct. No framework required.

---

## Key Design Decisions

### No binary dependencies

Both the Helm SDK and `client-go` are pure Go. This means the binary can run in a minimal container image (e.g., `scratch` or `distroless`) without shelling out.

### Typed identifiers

`deployment.ID`, `deployment.ReleaseName`, `deployment.Namespace` are distinct types, not plain `string`. This prevents subtle bugs like passing a namespace where a release name is expected.

### Pure-Go SQLite

`modernc.org/sqlite` compiles the SQLite amalgamation to Go via a transpiler, so there is no CGo dependency and no need for a C toolchain in CI or containers.

### Retry at the adapter boundary

Helm and K8s operations are retried with exponential back-off inside the adapters, keeping retry logic out of the application services. Policies are injectable so tests can use zero-delay retries.

### MCP as the only inbound port

There is no REST API for deployment operations. All external requests arrive via MCP tool calls, which means any MCP-compatible client (AI or otherwise) can drive the orchestrator.
