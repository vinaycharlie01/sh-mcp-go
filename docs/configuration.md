# Configuration Reference

Configuration is loaded from YAML files and can be overridden by environment variables. The lookup order (highest priority first):

1. Environment variables (prefix `SHMCP_`, `.` → `_`)
2. `./configs/config.yaml` (or path given to `--config`)
3. `$HOME/.sh-mcp-go/config.yaml`
4. `/etc/sh-mcp-go/config.yaml`
5. Built-in defaults (listed below)

---

## Server

Controls the HTTP server that serves the Prometheus `/metrics` endpoint.

```yaml
server:
  host: "0.0.0.0"          # bind address
  port: 8080                # listen port
  read_timeout: 30s         # max time to read a request
  write_timeout: 30s        # max time to write a response
  idle_timeout: 120s        # keep-alive idle timeout
  shutdown_timeout: 15s     # graceful shutdown window
```

| Key | Env | Default | Description |
|-----|-----|---------|-------------|
| `server.host` | `SHMCP_SERVER_HOST` | `0.0.0.0` | Bind address |
| `server.port` | `SHMCP_SERVER_PORT` | `8080` | Listen port |
| `server.read_timeout` | `SHMCP_SERVER_READ_TIMEOUT` | `30s` | Request read timeout |
| `server.write_timeout` | `SHMCP_SERVER_WRITE_TIMEOUT` | `30s` | Response write timeout |
| `server.idle_timeout` | `SHMCP_SERVER_IDLE_TIMEOUT` | `120s` | Keep-alive idle timeout |
| `server.shutdown_timeout` | `SHMCP_SERVER_SHUTDOWN_TIMEOUT` | `15s` | Graceful shutdown window |

---

## Kubernetes

```yaml
kubernetes:
  in_cluster: false                # true when running inside a pod
  kubeconfig: ""                   # path to kubeconfig (empty = $KUBECONFIG or ~/.kube/config)
  context: ""                      # kubeconfig context (empty = current context)
  qps: 50                          # client-go QPS throttle
  burst: 100                       # client-go burst limit
  timeout: 30s                     # default API call timeout
  default_namespace: "default"     # fallback namespace
```

| Key | Env | Default | Description |
|-----|-----|---------|-------------|
| `kubernetes.in_cluster` | `SHMCP_KUBERNETES_IN_CLUSTER` | `false` | Use in-cluster service account |
| `kubernetes.kubeconfig` | `SHMCP_KUBERNETES_KUBECONFIG` | `""` | Path to kubeconfig file |
| `kubernetes.context` | `SHMCP_KUBERNETES_CONTEXT` | `""` | kubeconfig context name |
| `kubernetes.qps` | `SHMCP_KUBERNETES_QPS` | `50` | API request rate limit |
| `kubernetes.burst` | `SHMCP_KUBERNETES_BURST` | `100` | Burst above QPS |
| `kubernetes.timeout` | `SHMCP_KUBERNETES_TIMEOUT` | `30s` | Default operation timeout |
| `kubernetes.default_namespace` | `SHMCP_KUBERNETES_DEFAULT_NAMESPACE` | `default` | Fallback namespace |

---

## Helm

```yaml
helm:
  repository_cache: "/tmp/sh-mcp-go/helm/cache"
  repository_config: "/tmp/sh-mcp-go/helm/repositories.yaml"
  default_timeout: 300s      # 5 minutes
  max_history: 10            # max stored revisions per release
  atomic: true               # roll back on failure
  wait_for_jobs: true        # wait for Job completion
```

| Key | Env | Default | Description |
|-----|-----|---------|-------------|
| `helm.repository_cache` | `SHMCP_HELM_REPOSITORY_CACHE` | `/tmp/helm/cache` | Local chart cache directory |
| `helm.repository_config` | `SHMCP_HELM_REPOSITORY_CONFIG` | `/tmp/helm/repositories.yaml` | Repository index file |
| `helm.default_timeout` | `SHMCP_HELM_DEFAULT_TIMEOUT` | `5m` | Default install/upgrade timeout |
| `helm.max_history` | `SHMCP_HELM_MAX_HISTORY` | `10` | Max revision history per release |
| `helm.atomic` | `SHMCP_HELM_ATOMIC` | `true` | Auto-rollback on failure |
| `helm.wait_for_jobs` | `SHMCP_HELM_WAIT_FOR_JOBS` | `true` | Block until Job pods complete |

---

## Storage

```yaml
storage:
  driver: "sqlite"      # currently only sqlite is supported
  sqlite:
    path: "/tmp/sh-mcp-go/state.db"
```

| Key | Env | Default | Description |
|-----|-----|---------|-------------|
| `storage.driver` | `SHMCP_STORAGE_DRIVER` | `sqlite` | Storage backend |
| `storage.sqlite.path` | `SHMCP_STORAGE_SQLITE_PATH` | `/var/lib/sh-mcp-go/state.db` | SQLite database file path |

The SQLite database is created automatically on first run. For production deployments, map the path to a persistent volume.

---

## Observability

```yaml
observability:
  metrics_enabled: true
  tracing_enabled: false
  otlp_endpoint: "localhost:4318"   # OTLP HTTP endpoint (used when tracing_enabled: true)
  service_name: "sh-mcp-go"
  sampling_rate: 0.1                # 10% trace sampling
```

| Key | Env | Default | Description |
|-----|-----|---------|-------------|
| `observability.metrics_enabled` | `SHMCP_OBSERVABILITY_METRICS_ENABLED` | `true` | Expose Prometheus `/metrics` |
| `observability.tracing_enabled` | `SHMCP_OBSERVABILITY_TRACING_ENABLED` | `false` | Enable OTLP tracing export |
| `observability.otlp_endpoint` | `SHMCP_OBSERVABILITY_OTLP_ENDPOINT` | `localhost:4318` | OTLP collector endpoint |
| `observability.service_name` | `SHMCP_OBSERVABILITY_SERVICE_NAME` | `sh-mcp-go` | Service name in traces/metrics |
| `observability.sampling_rate` | `SHMCP_OBSERVABILITY_SAMPLING_RATE` | `0.1` | Fraction of traces to export |

### Enabling tracing

```yaml
observability:
  tracing_enabled: true
  otlp_endpoint: "otel-collector:4318"
```

Traces are exported via OTLP/HTTP. Any OpenTelemetry-compatible backend (Jaeger, Tempo, Honeycomb, Datadog) works.

---

## Security

```yaml
security:
  enable_rbac_validation: false     # validate RBAC before deploys
  enable_secret_masking: true       # mask secret values in logs
  denied_namespaces:
    - "kube-system"
    - "kube-public"
```

| Key | Env | Default | Description |
|-----|-----|---------|-------------|
| `security.enable_rbac_validation` | `SHMCP_SECURITY_ENABLE_RBAC_VALIDATION` | `false` | Check RBAC permissions before operations |
| `security.enable_secret_masking` | `SHMCP_SECURITY_ENABLE_SECRET_MASKING` | `true` | Redact secret values in structured logs |
| `security.denied_namespaces` | — | `[]` | Namespaces that cannot be targeted |

---

## MCP

```yaml
mcp:
  transport: "stdio"         # "stdio" or "sse"
  name: "sh-mcp-go"
  version: "1.0.0"
  sse_addr: "0.0.0.0:8081"  # address for SSE transport
```

| Key | Env | Default | Description |
|-----|-----|---------|-------------|
| `mcp.transport` | `SHMCP_MCP_TRANSPORT` | `stdio` | Transport protocol (`stdio` or `sse`) |
| `mcp.name` | `SHMCP_MCP_NAME` | `sh-mcp-go` | Server name advertised to clients |
| `mcp.version` | `SHMCP_MCP_VERSION` | `1.0.0` | Server version advertised to clients |
| `mcp.sse_addr` | `SHMCP_MCP_SSE_ADDR` | `0.0.0.0:8081` | Bind address for SSE transport |

### Transport modes

**`stdio`** — The MCP server reads from stdin and writes to stdout. This is the standard mode for Claude Desktop and other local AI clients that spawn the server as a subprocess.

**`sse`** — The MCP server listens on `sse_addr` and clients connect over HTTP Server-Sent Events. Use this for remote or multi-client setups.

---

## Logging

```yaml
log:
  level: "info"    # debug | info | warn | error
  format: "json"   # json | text
```

| Key | Env | Default | Description |
|-----|-----|---------|-------------|
| `log.level` | `SHMCP_LOG_LEVEL` | `info` | Log verbosity |
| `log.format` | `SHMCP_LOG_FORMAT` | `json` | Output format (`json` for production, `text` for development) |

---

## Full Example

```yaml
server:
  host: "0.0.0.0"
  port: 8080
  shutdown_timeout: 15s

kubernetes:
  in_cluster: true
  default_namespace: "default"
  qps: 100
  burst: 200

helm:
  repository_cache: "/var/cache/helm"
  repository_config: "/var/cache/helm/repositories.yaml"
  default_timeout: 600s
  max_history: 20
  atomic: true

storage:
  driver: "sqlite"
  sqlite:
    path: "/data/sh-mcp-go/state.db"

observability:
  metrics_enabled: true
  tracing_enabled: true
  otlp_endpoint: "otel-collector.monitoring.svc.cluster.local:4318"
  service_name: "sh-mcp-go"
  sampling_rate: 0.25

security:
  enable_rbac_validation: true
  enable_secret_masking: true
  denied_namespaces:
    - "kube-system"
    - "kube-public"
    - "kube-node-lease"

mcp:
  transport: "sse"
  sse_addr: "0.0.0.0:8081"

log:
  level: "info"
  format: "json"
```
