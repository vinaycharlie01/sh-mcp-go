# MCP Tools Reference

All 24 tools registered by `sh-mcp-go`. Parameters marked **required** must be provided; all others are optional with the default shown.

---

## Chart Lifecycle

### `install_chart`

Install a Helm chart onto a Kubernetes cluster.

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `release_name` | string | yes | — | Helm release name (lowercase alphanumeric, hyphens allowed) |
| `chart_name` | string | yes | — | Chart name as it appears in the repository |
| `repo_url` | string | yes | — | Helm repository URL |
| `namespace` | string | | `"default"` | Target Kubernetes namespace |
| `version` | string | | latest | Chart version (semver) |
| `values` | object | | `{}` | Values overrides (equivalent to `-f values.yaml`) |
| `dry_run` | boolean | | `false` | Perform a dry run without applying changes |
| `wait` | boolean | | `true` | Wait for all resources to become ready |
| `atomic` | boolean | | `true` | Roll back automatically on failure |
| `create_namespace` | boolean | | `true` | Create the namespace if it doesn't exist |
| `timeout_seconds` | number | | `300` | Operation timeout in seconds |

**Example:**

```json
{
  "release_name": "prometheus",
  "chart_name": "kube-prometheus-stack",
  "repo_url": "https://prometheus-community.github.io/helm-charts",
  "namespace": "monitoring",
  "version": "65.1.1",
  "values": {
    "grafana": { "enabled": true },
    "prometheus": { "prometheusSpec": { "retention": "15d" } }
  },
  "create_namespace": true,
  "timeout_seconds": 600
}
```

---

### `upgrade_chart`

Upgrade an existing Helm release to a new chart version or with new values.

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `release_name` | string | yes | — | Existing release name |
| `chart_name` | string | yes | — | Chart name |
| `repo_url` | string | yes | — | Helm repository URL |
| `namespace` | string | | `"default"` | Release namespace |
| `version` | string | | latest | Target chart version |
| `values` | object | | `{}` | Values overrides |
| `dry_run` | boolean | | `false` | Perform a dry run |
| `wait` | boolean | | `true` | Wait for readiness |
| `atomic` | boolean | | `true` | Roll back on failure |
| `reuse_values` | boolean | | `false` | Reuse existing release values |
| `reset_values` | boolean | | `false` | Reset values to chart defaults before applying overrides |

---

### `rollback_chart`

Roll back a Helm release to a previous revision.

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `release_name` | string | yes | — | Release name to roll back |
| `namespace` | string | yes | — | Release namespace |
| `version` | number | | `0` | Target revision number (`0` = previous revision) |
| `dry_run` | boolean | | `false` | Perform a dry run |
| `wait` | boolean | | `true` | Wait for readiness after rollback |

---

### `uninstall_chart`

Uninstall a Helm release from the cluster.

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `release_name` | string | yes | — | Release name to uninstall |
| `namespace` | string | yes | — | Release namespace |
| `dry_run` | boolean | | `false` | Perform a dry run |
| `keep_history` | boolean | | `false` | Retain release history after uninstall |

---

## Operator Lifecycle

### `install_operator`

Install a Kubernetes operator via OLM or Helm-based operator chart.

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `name` | string | yes | — | Operator name |
| `namespace` | string | | `"operators"` | Installation namespace |
| `channel` | string | | `"stable"` | OLM subscription channel |
| `source` | string | | `"operatorhubio-catalog"` | Catalog source name |

---

### `upgrade_operator`

Upgrade an installed Kubernetes operator to a new channel or version.

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `name` | string | yes | — | Operator name |
| `namespace` | string | yes | — | Operator namespace |
| `channel` | string | | — | New channel to subscribe to |
| `version` | string | | — | Target version |

---

### `rollback_operator`

Roll back an operator upgrade.

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `name` | string | yes | — | Operator name |
| `namespace` | string | yes | — | Operator namespace |

---

### `delete_operator`

Delete an installed Kubernetes operator.

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `name` | string | yes | — | Operator name |
| `namespace` | string | yes | — | Operator namespace |

---

## Planning

### `plan_deployment`

Generate an AI deployment plan from a natural language intent. Returns an ordered list of steps with a dependency graph.

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `intent` | string | yes | — | Natural language description, e.g. `"Deploy Prometheus and Grafana with persistent storage"` |
| `namespace` | string | | `"default"` | Target namespace for the deployment |

**Response fields:**

| Field | Description |
|-------|-------------|
| `plan_id` | Unique plan identifier |
| `intent` | The original intent string |
| `steps` | Ordered list of steps with `id`, `type`, `description`, `params`, `depends_on` |
| `warnings` | Any warnings generated during planning |
| `estimated_mins` | Estimated execution time in minutes |
| `rollback_plan` | Companion rollback plan (steps to undo the deployment) |

**Step types:** `CREATE_NAMESPACE`, `INSTALL_CRDS`, `INSTALL_DEPENDENCY`, `INSTALL_CHART`, `UPGRADE_CHART`, `ROLLBACK_CHART`, `UNINSTALL_CHART`, `INSTALL_OPERATOR`, `VALIDATE_READINESS`, `CONFIGURE_INGRESS`, `CONFIGURE_STORAGE`, `RUN_HEALTH_CHECK`, `RUN_SECURITY_SCAN`

**Example:**

```json
{
  "intent": "Deploy a PostgreSQL cluster with CloudNativePG operator and automated backups to S3",
  "namespace": "databases"
}
```

---

## Validation

### `validate_cluster`

Run prerequisite checks on the cluster to ensure it is ready for deployments. No parameters.

Checks include: API server reachability, node readiness, storage class availability, required CRDs.

---

### `validate_release`

Validate an existing Helm release and its resource health.

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `release_name` | string | yes | — | Release name |
| `namespace` | string | yes | — | Release namespace |

---

## Inventory

### `cluster_inventory`

Return a complete inventory of the cluster: nodes, namespaces, Helm releases, CRDs. No parameters.

---

### `release_inventory`

List all Helm releases, optionally filtered by namespace.

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `namespace` | string | | — | Filter by namespace (omit for all namespaces) |

**Response fields:** `releases` (array), `count` (integer). Each release has `name`, `namespace`, `chart`, `app_version`, `status`, `revision`.

---

## Analysis

### `resource_estimation`

Estimate CPU, memory, and storage requirements for a given chart deployment.

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `chart_name` | string | yes | — | Chart name |
| `namespace` | string | | `"default"` | Target namespace |
| `replicas` | number | | `1` | Number of replicas |

---

### `dependency_analysis`

Analyse chart dependencies and identify required CRDs, operators, and charts that must be installed first.

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `chart_name` | string | yes | — | Chart name |
| `repo_url` | string | yes | — | Helm repository URL |
| `version` | string | | latest | Chart version |

---

### `security_scan`

Scan a Helm release or chart for security issues: RBAC over-permission, privilege escalation, image vulnerabilities.

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `release_name` | string | | — | Release name to scan (post-install scan) |
| `namespace` | string | | `"default"` | Release namespace |
| `chart_name` | string | | — | Chart name for pre-install scan |
| `repo_url` | string | | — | Repository URL for pre-install scan |

---

### `health_check`

Check the health of all resources in a Helm release.

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `release_name` | string | yes | — | Release name |
| `namespace` | string | yes | — | Release namespace |

**Response fields:** `release_name`, `namespace`, `healthy` (bool), `resources` (array with per-resource `ready` status).

---

### `generate_rca`

Generate a root cause analysis for a failing or degraded Helm release.

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `release_name` | string | yes | — | Release name |
| `namespace` | string | yes | — | Release namespace |

Returns a human-readable analysis string describing likely causes and remediation steps.

---

### `analyze_failure`

Analyse a deployment failure and suggest remediation steps.

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `release_name` | string | yes | — | Failed release name |
| `namespace` | string | yes | — | Release namespace |
| `error_message` | string | | — | Error message from the failed deployment |

Returns a structured failure analysis with root cause and concrete `kubectl` commands to investigate further.

---

## Recommendations

### `generate_values_yaml`

Generate a values.yaml skeleton for a Helm chart with defaults and annotations.

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `chart_name` | string | yes | — | Chart name |
| `repo_url` | string | yes | — | Repository URL |
| `version` | string | | latest | Chart version |

---

### `recommend_upgrade`

Recommend whether and how to upgrade a Helm release, comparing current vs. latest chart version.

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `release_name` | string | yes | — | Release name |
| `namespace` | string | yes | — | Release namespace |

**Response fields:** `release_name`, `current_version`, `latest_version`, `upgrade_recommended` (bool), `notes`.

---

### `recommend_operator`

Recommend a Kubernetes operator for a given workload type.

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `workload_type` | string | yes | — | One of: `database`, `messaging`, `monitoring`, `storage` |

Returns a list of recommended operators with chart name and repository URL.

**Example:**

```json
{ "workload_type": "database" }
```

Response:

```json
{
  "workload_type": "database",
  "recommendations": [
    {
      "name": "CloudNativePG",
      "description": "PostgreSQL operator",
      "chart": "cloudnative-pg",
      "repo": "https://cloudnative-pg.io/charts"
    },
    {
      "name": "MySQL Operator",
      "description": "MySQL/InnoDB cluster",
      "chart": "mysql-operator",
      "repo": "https://mysql.github.io/mysql-operator/"
    }
  ]
}
```

---

## Status

### `deployment_status`

Get the current status of a deployment tracked by `sh-mcp-go`.

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `deployment_id` | string | | — | Internal deployment ID |
| `release_name` | string | | — | Release name (alternative to `deployment_id`) |
| `namespace` | string | | `"default"` | Release namespace |

**Response fields:** `release_name`, `namespace`, `revision`, `status`, `chart`, `chart_version`, `deployed_at`.

---

### `release_status`

Get the current status of a Helm release directly from the cluster.

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `release_name` | string | yes | — | Release name |
| `namespace` | string | yes | — | Release namespace |

Same response shape as `deployment_status`.
