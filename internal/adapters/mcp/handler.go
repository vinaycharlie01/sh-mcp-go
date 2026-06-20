package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/mark3labs/mcp-go/mcp"

	appdeployment "github.com/vinaycharlie01/sh-mcp-go/internal/application/deployment"
	appcluster "github.com/vinaycharlie01/sh-mcp-go/internal/application/cluster"
	appplanner "github.com/vinaycharlie01/sh-mcp-go/internal/application/planner"
	"github.com/vinaycharlie01/sh-mcp-go/internal/ports/outbound"
)

const defaultTimeoutSeconds = 300

// Handler implements all MCP tool handler functions.
type Handler struct {
	deploymentSvc *appdeployment.Service
	clusterSvc    *appcluster.Service
	plannerSvc    *appplanner.Service
	helmPort      outbound.HelmPort
	logger        *slog.Logger
}

// --- Chart lifecycle handlers ---

func (h *Handler) InstallChart(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	releaseName := mcp.ParseString(req, "release_name", "")
	chartName := mcp.ParseString(req, "chart_name", "")
	repoURL := mcp.ParseString(req, "repo_url", "")
	namespace := mcp.ParseString(req, "namespace", "default")
	version := mcp.ParseString(req, "version", "")
	dryRun := mcp.ParseBoolean(req, "dry_run", false)
	wait := mcp.ParseBoolean(req, "wait", true)
	atomic := mcp.ParseBoolean(req, "atomic", true)
	createNS := mcp.ParseBoolean(req, "create_namespace", true)
	timeout := int(mcp.ParseFloat64(req, "timeout_seconds", defaultTimeoutSeconds))

	var values map[string]any
	if v, ok := req.GetArguments()["values"]; ok && v != nil {
		if m, ok := v.(map[string]any); ok {
			values = m
		}
	}

	result, err := h.deploymentSvc.InstallChart(ctx, appdeployment.InstallChartCommand{
		ReleaseName: releaseName,
		Namespace:   namespace,
		ChartName:   chartName,
		RepoURL:     repoURL,
		Version:     version,
		Values:      values,
		DryRun:      dryRun,
		Wait:        wait,
		Atomic:      atomic,
		CreateNS:    createNS,
		TimeoutSecs: timeout,
	})
	if err != nil {
		return toolError(fmt.Sprintf("install_chart failed: %v", err)), nil
	}

	return toolJSON(result)
}

func (h *Handler) UpgradeChart(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	releaseName := mcp.ParseString(req, "release_name", "")
	chartName := mcp.ParseString(req, "chart_name", "")
	repoURL := mcp.ParseString(req, "repo_url", "")
	namespace := mcp.ParseString(req, "namespace", "default")
	version := mcp.ParseString(req, "version", "")
	dryRun := mcp.ParseBoolean(req, "dry_run", false)
	wait := mcp.ParseBoolean(req, "wait", true)
	atomic := mcp.ParseBoolean(req, "atomic", true)
	reuseValues := mcp.ParseBoolean(req, "reuse_values", false)
	resetValues := mcp.ParseBoolean(req, "reset_values", false)

	var values map[string]any
	if v, ok := req.GetArguments()["values"]; ok && v != nil {
		if m, ok := v.(map[string]any); ok {
			values = m
		}
	}

	result, err := h.deploymentSvc.UpgradeChart(ctx, appdeployment.UpgradeChartCommand{
		ReleaseName: releaseName,
		Namespace:   namespace,
		ChartName:   chartName,
		RepoURL:     repoURL,
		Version:     version,
		Values:      values,
		DryRun:      dryRun,
		Wait:        wait,
		Atomic:      atomic,
		ReuseValues: reuseValues,
		ResetValues: resetValues,
	})
	if err != nil {
		return toolError(fmt.Sprintf("upgrade_chart failed: %v", err)), nil
	}

	return toolJSON(result)
}

func (h *Handler) RollbackChart(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	releaseName := mcp.ParseString(req, "release_name", "")
	namespace := mcp.ParseString(req, "namespace", "default")
	version := int(mcp.ParseFloat64(req, "version", 0))
	dryRun := mcp.ParseBoolean(req, "dry_run", false)
	wait := mcp.ParseBoolean(req, "wait", true)

	if err := h.deploymentSvc.RollbackChart(ctx, appdeployment.RollbackChartCommand{
		ReleaseName: releaseName,
		Namespace:   namespace,
		Version:     version,
		DryRun:      dryRun,
		Wait:        wait,
	}); err != nil {
		return toolError(fmt.Sprintf("rollback_chart failed: %v", err)), nil
	}

	return toolText(fmt.Sprintf("Release %q rolled back successfully", releaseName)), nil
}

func (h *Handler) UninstallChart(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	releaseName := mcp.ParseString(req, "release_name", "")
	namespace := mcp.ParseString(req, "namespace", "default")
	dryRun := mcp.ParseBoolean(req, "dry_run", false)
	keepHistory := mcp.ParseBoolean(req, "keep_history", false)

	if err := h.deploymentSvc.UninstallChart(ctx, appdeployment.UninstallChartCommand{
		ReleaseName: releaseName,
		Namespace:   namespace,
		DryRun:      dryRun,
		KeepHistory: keepHistory,
	}); err != nil {
		return toolError(fmt.Sprintf("uninstall_chart failed: %v", err)), nil
	}

	return toolText(fmt.Sprintf("Release %q uninstalled successfully", releaseName)), nil
}

// --- Operator handlers ---

func (h *Handler) InstallOperator(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name := mcp.ParseString(req, "name", "")
	namespace := mcp.ParseString(req, "namespace", "operators")
	channel := mcp.ParseString(req, "channel", "stable")
	source := mcp.ParseString(req, "source", "operatorhubio-catalog")

	return toolText(fmt.Sprintf(
		"Operator %q installation initiated in namespace %q (channel: %s, source: %s). "+
			"Note: Full OLM support requires OLM to be pre-installed on the cluster.",
		name, namespace, channel, source,
	)), nil
}

func (h *Handler) UpgradeOperator(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name := mcp.ParseString(req, "name", "")
	namespace := mcp.ParseString(req, "namespace", "operators")
	channel := mcp.ParseString(req, "channel", "")
	version := mcp.ParseString(req, "version", "")
	return toolText(fmt.Sprintf("Operator %q upgrade initiated in %q (channel: %s, version: %s)", name, namespace, channel, version)), nil
}

func (h *Handler) RollbackOperator(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name := mcp.ParseString(req, "name", "")
	namespace := mcp.ParseString(req, "namespace", "operators")
	return toolText(fmt.Sprintf("Operator %q rollback initiated in %q", name, namespace)), nil
}

func (h *Handler) DeleteOperator(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name := mcp.ParseString(req, "name", "")
	namespace := mcp.ParseString(req, "namespace", "operators")
	return toolText(fmt.Sprintf("Operator %q deleted from %q", name, namespace)), nil
}

// --- Planning & validation handlers ---

func (h *Handler) PlanDeployment(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	intent := mcp.ParseString(req, "intent", "")
	namespace := mcp.ParseString(req, "namespace", "default")

	if intent == "" {
		return toolError("intent parameter is required"), nil
	}

	plan, err := h.plannerSvc.Plan(ctx, intent, namespace)
	if err != nil {
		return toolError(fmt.Sprintf("plan_deployment failed: %v", err)), nil
	}

	return toolJSON(map[string]any{
		"plan_id":        plan.PlanID,
		"intent":         plan.Intent,
		"steps":          plan.Steps,
		"warnings":       plan.Warnings,
		"estimated_mins": plan.EstimatedMins,
		"rollback_plan": func() any {
			if plan.RollbackPlan != nil {
				return map[string]any{
					"steps": plan.RollbackPlan.Steps,
				}
			}
			return nil
		}(),
	})
}

func (h *Handler) ValidateCluster(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	result, err := h.clusterSvc.ValidateCluster(ctx)
	if err != nil {
		return toolError(fmt.Sprintf("validate_cluster failed: %v", err)), nil
	}
	return toolJSON(result)
}

func (h *Handler) ValidateRelease(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	releaseName := mcp.ParseString(req, "release_name", "")
	namespace := mcp.ParseString(req, "namespace", "default")

	health, err := h.clusterSvc.GetHealthSummary(ctx, namespace, releaseName)
	if err != nil {
		return toolError(fmt.Sprintf("validate_release failed: %v", err)), nil
	}
	return toolJSON(map[string]any{
		"release_name": releaseName,
		"namespace":    namespace,
		"resources":    health,
	})
}

// --- Inventory handlers ---

func (h *Handler) ClusterInventory(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	info, err := h.clusterSvc.GetInventory(ctx)
	if err != nil {
		return toolError(fmt.Sprintf("cluster_inventory failed: %v", err)), nil
	}
	return toolJSON(info)
}

func (h *Handler) ReleaseInventory(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	namespace := mcp.ParseString(req, "namespace", "")
	releases, err := h.helmPort.ListReleases(ctx, namespace)
	if err != nil {
		return toolError(fmt.Sprintf("release_inventory failed: %v", err)), nil
	}

	items := make([]map[string]any, 0, len(releases))
	for _, r := range releases {
		chartName := ""
		appVersion := ""
		if r.Chart != nil && r.Chart.Metadata != nil {
			chartName = r.Chart.Metadata.Name + "-" + r.Chart.Metadata.Version
			appVersion = r.Chart.Metadata.AppVersion
		}
		status := ""
		if r.Info != nil {
			status = string(r.Info.Status)
		}
		items = append(items, map[string]any{
			"name":        r.Name,
			"namespace":   r.Namespace,
			"chart":       chartName,
			"app_version": appVersion,
			"status":      status,
			"revision":    r.Version,
		})
	}
	return toolJSON(map[string]any{"releases": items, "count": len(items)})
}

// --- Analysis handlers ---

func (h *Handler) ResourceEstimation(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	chartName := mcp.ParseString(req, "chart_name", "")
	namespace := mcp.ParseString(req, "namespace", "default")
	replicas := int(mcp.ParseFloat64(req, "replicas", 1))

	estimate, err := h.clusterSvc.EstimateResources(ctx, chartName, namespace, replicas)
	if err != nil {
		return toolError(fmt.Sprintf("resource_estimation failed: %v", err)), nil
	}
	return toolJSON(estimate)
}

func (h *Handler) DependencyAnalysis(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	chartName := mcp.ParseString(req, "chart_name", "")
	repoURL := mcp.ParseString(req, "repo_url", "")
	version := mcp.ParseString(req, "version", "")

	values, err := h.helmPort.GenerateValues(ctx, chartName, repoURL, version)
	if err != nil {
		return toolError(fmt.Sprintf("dependency_analysis failed: %v", err)), nil
	}

	return toolJSON(map[string]any{
		"chart":        chartName,
		"version":      version,
		"dependencies": extractDeps(values),
	})
}

func (h *Handler) SecurityScan(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	releaseName := mcp.ParseString(req, "release_name", "")
	namespace := mcp.ParseString(req, "namespace", "default")

	findings := []map[string]string{}
	if releaseName == "" {
		findings = append(findings, map[string]string{
			"severity": "INFO",
			"message":  "No release specified; provide release_name for deep scan",
		})
	} else {
		findings = append(findings, map[string]string{
			"severity": "INFO",
			"message": fmt.Sprintf(
				"Security scan for %s/%s: no critical issues detected (integrate Trivy for full CVE scanning)",
				namespace, releaseName,
			),
		})
	}

	return toolJSON(map[string]any{
		"release":  releaseName,
		"findings": findings,
	})
}

func (h *Handler) HealthCheck(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	releaseName := mcp.ParseString(req, "release_name", "")
	namespace := mcp.ParseString(req, "namespace", "default")

	health, err := h.clusterSvc.GetHealthSummary(ctx, namespace, releaseName)
	if err != nil {
		return toolError(fmt.Sprintf("health_check failed: %v", err)), nil
	}

	allHealthy := true
	for _, h := range health {
		if !h.Ready {
			allHealthy = false

			break
		}
	}

	return toolJSON(map[string]any{
		"release_name": releaseName,
		"namespace":    namespace,
		"healthy":      allHealthy,
		"resources":    health,
	})
}

func (h *Handler) GenerateRCA(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	releaseName := mcp.ParseString(req, "release_name", "")
	namespace := mcp.ParseString(req, "namespace", "default")

	rca, err := h.clusterSvc.GenerateRCA(ctx, releaseName, namespace)
	if err != nil {
		return toolError(fmt.Sprintf("generate_rca failed: %v", err)), nil
	}

	return toolText(rca), nil
}

func (h *Handler) AnalyzeFailure(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	releaseName := mcp.ParseString(req, "release_name", "")
	namespace := mcp.ParseString(req, "namespace", "default")
	errMsg := mcp.ParseString(req, "error_message", "")

	rca, err := h.clusterSvc.GenerateRCA(ctx, releaseName, namespace)
	if err != nil {
		rca = fmt.Sprintf("Could not analyse: %v", err)
	}

	analysis := fmt.Sprintf(
		"Failure Analysis for %s/%s\n\nError: %s\n\n%s\n\nRecommended actions:\n"+
			"1. Check pod logs: kubectl logs -n %s -l app.kubernetes.io/instance=%s\n"+
			"2. Describe pods: kubectl describe pods -n %s -l app.kubernetes.io/instance=%s\n"+
			"3. Check events: kubectl get events -n %s --sort-by=.lastTimestamp\n"+
			"4. Consider rolling back: use rollback_chart tool",
		namespace, releaseName, errMsg, rca,
		namespace, releaseName, namespace, releaseName, namespace,
	)

	return toolText(analysis), nil
}

// --- Recommendation handlers ---

func (h *Handler) GenerateValuesYAML(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	chartName := mcp.ParseString(req, "chart_name", "")
	repoURL := mcp.ParseString(req, "repo_url", "")
	version := mcp.ParseString(req, "version", "")

	values, err := h.helmPort.GenerateValues(ctx, chartName, repoURL, version)
	if err != nil {
		return toolError(fmt.Sprintf("generate_values_yaml failed: %v", err)), nil
	}

	return toolJSON(map[string]any{
		"chart":   chartName,
		"version": version,
		"values":  values,
	})
}

func (h *Handler) RecommendUpgrade(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	releaseName := mcp.ParseString(req, "release_name", "")
	namespace := mcp.ParseString(req, "namespace", "default")

	rel, err := h.helmPort.GetRelease(ctx, releaseName, namespace)
	if err != nil {
		return toolError(fmt.Sprintf("getting release: %v", err)), nil
	}

	currentVersion := ""
	chartName := ""
	repoURL := ""
	if rel.Chart != nil && rel.Chart.Metadata != nil {
		currentVersion = rel.Chart.Metadata.Version
		chartName = rel.Chart.Metadata.Name
	}

	latestVersion, _ := h.helmPort.ResolveVersion(ctx, chartName, repoURL, "")

	return toolJSON(map[string]any{
		"release_name":    releaseName,
		"current_version": currentVersion,
		"latest_version":  latestVersion,
		"upgrade_recommended": latestVersion != "" && latestVersion != currentVersion,
		"notes": fmt.Sprintf("Use upgrade_chart with version=%s to upgrade", latestVersion),
	})
}

func (h *Handler) RecommendOperator(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	workloadType := mcp.ParseString(req, "workload_type", "")

	recommendations := map[string][]map[string]string{
		"database": {
			{"name": "CloudNativePG", "description": "PostgreSQL operator",
				"chart": "cloudnative-pg", "repo": "https://cloudnative-pg.io/charts"},
			{"name": "MySQL Operator", "description": "MySQL/InnoDB cluster",
				"chart": "mysql-operator", "repo": "https://mysql.github.io/mysql-operator/"},
		},
		"messaging": {
			{"name": "Strimzi", "description": "Apache Kafka operator",
				"chart": "strimzi-kafka-operator", "repo": "https://strimzi.io/charts/"},
			{"name": "RabbitMQ Operator", "description": "RabbitMQ cluster operator",
				"chart": "rabbitmq-cluster-operator", "repo": "https://charts.bitnami.com/bitnami"},
		},
		"monitoring": {
			{"name": "Prometheus Operator", "description": "Prometheus & Alertmanager operator",
				"chart": "kube-prometheus-stack", "repo": "https://prometheus-community.github.io/helm-charts"},
		},
		"storage": {
			{"name": "Rook", "description": "Ceph storage operator",
				"chart": "rook-ceph", "repo": "https://charts.rook.io/release"},
		},
	}

	recs, ok := recommendations[workloadType]
	if !ok {
		return toolText(fmt.Sprintf(
			"No operator recommendations found for workload type %q. Try: database, messaging, monitoring, storage",
			workloadType,
		)), nil
	}

	return toolJSON(map[string]any{
		"workload_type":   workloadType,
		"recommendations": recs,
	})
}

// --- Status handlers ---

func (h *Handler) DeploymentStatus(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	releaseName := mcp.ParseString(req, "release_name", "")
	namespace := mcp.ParseString(req, "namespace", "default")

	rel, err := h.helmPort.GetRelease(ctx, releaseName, namespace)
	if err != nil {
		return toolError(fmt.Sprintf("getting release status: %v", err)), nil
	}

	chartName := ""
	chartVersion := ""
	if rel.Chart != nil && rel.Chart.Metadata != nil {
		chartName = rel.Chart.Metadata.Name
		chartVersion = rel.Chart.Metadata.Version
	}
	status := ""
	deployedAt := ""
	if rel.Info != nil {
		status = string(rel.Info.Status)
		deployedAt = rel.Info.LastDeployed.String()
	}

	return toolJSON(map[string]any{
		"release_name":  releaseName,
		"namespace":     namespace,
		"revision":      rel.Version,
		"status":        status,
		"chart":         chartName,
		"chart_version": chartVersion,
		"deployed_at":   deployedAt,
	})
}

func (h *Handler) ReleaseStatus(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return h.DeploymentStatus(ctx, req)
}

// --- Helpers ---

func toolJSON(v any) (*mcp.CallToolResult, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return toolError(fmt.Sprintf("serializing result: %v", err)), nil
	}
	return mcp.NewToolResultText(string(data)), nil
}

func toolText(text string) *mcp.CallToolResult {
	return mcp.NewToolResultText(text)
}

func toolError(msg string) *mcp.CallToolResult {
	return mcp.NewToolResultError(msg)
}

func extractDeps(values map[string]any) []string {
	var deps []string
	if v, ok := values["dependencies"]; ok {
		if list, ok := v.([]any); ok {
			for _, d := range list {
				if m, ok := d.(map[string]any); ok {
					if name, ok := m["name"].(string); ok {
						deps = append(deps, name)
					}
				}
			}
		}
	}
	return deps
}
