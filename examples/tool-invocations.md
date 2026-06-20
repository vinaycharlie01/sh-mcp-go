# Example Tool Invocations

These examples show how to phrase requests to an MCP-connected AI assistant. The assistant translates your intent into tool calls automatically.

---

## Deploying a monitoring stack

**You:** Deploy Prometheus and Grafana in the monitoring namespace with 15-day retention and persistent storage.

**What the AI does:**
1. Calls `plan_deployment` with intent = `"Deploy Prometheus and Grafana in the monitoring namespace with 15-day retention and persistent storage"`
2. Reviews the generated plan (namespace creation → chart install → readiness validation)
3. Calls `install_chart` for `kube-prometheus-stack`

```json
{
  "release_name": "prometheus",
  "chart_name": "kube-prometheus-stack",
  "repo_url": "https://prometheus-community.github.io/helm-charts",
  "namespace": "monitoring",
  "values": {
    "grafana": {
      "enabled": true,
      "persistence": { "enabled": true, "size": "10Gi" }
    },
    "prometheus": {
      "prometheusSpec": {
        "retention": "15d",
        "storageSpec": {
          "volumeClaimTemplate": {
            "spec": { "resources": { "requests": { "storage": "50Gi" } } }
          }
        }
      }
    }
  },
  "create_namespace": true,
  "timeout_seconds": 600
}
```

---

## Checking cluster health before a release

**You:** Is the cluster ready for a production deployment?

**What the AI does:**

1. Calls `validate_cluster` — checks nodes, storage classes, API availability
2. Calls `cluster_inventory` — lists nodes, namespaces, existing releases
3. Summarises findings and flags any issues

---

## Rolling back a failed upgrade

**You:** The Redis upgrade in the data namespace failed. Roll it back.

**What the AI does:**

1. Calls `generate_rca` → `{ "release_name": "redis", "namespace": "data" }` to understand the failure
2. Calls `rollback_chart` → `{ "release_name": "redis", "namespace": "data", "version": 0 }` (0 = previous revision)
3. Calls `health_check` → `{ "release_name": "redis", "namespace": "data" }` to confirm recovery

---

## Getting an operator recommendation

**You:** What operator should I use for running PostgreSQL on Kubernetes?

**What the AI does:**

Calls `recommend_operator`:

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
    }
  ]
}
```

---

## Estimating resources before install

**You:** How much CPU and memory will the ELK stack need if I run 3 replicas?

**What the AI does:**

Calls `resource_estimation`:

```json
{
  "chart_name": "elasticsearch",
  "namespace": "logging",
  "replicas": 3
}
```

---

## Generating a values file

**You:** Give me a values.yaml for the latest Cert-Manager chart so I can customise it.

**What the AI does:**

Calls `generate_values_yaml`:

```json
{
  "chart_name": "cert-manager",
  "repo_url": "https://charts.jetstack.io"
}
```

The AI returns the full default values so you can override only what you need.

---

## Investigating a failure

**You:** The nginx-ingress install in production failed. What went wrong?

**What the AI does:**

1. Calls `analyze_failure`:

```json
{
  "release_name": "nginx-ingress",
  "namespace": "production",
  "error_message": "timed out waiting for the condition"
}
```

2. Returns a structured analysis: root cause (e.g. insufficient node resources, image pull failure), exact `kubectl` commands to investigate further, and a recommendation to roll back or retry.
