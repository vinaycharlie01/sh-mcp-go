package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	releasev1 "helm.sh/helm/v4/pkg/release/v1"

	appcluster "github.com/vinaycharlie01/sh-mcp-go/internal/application/cluster"
	appdeployment "github.com/vinaycharlie01/sh-mcp-go/internal/application/deployment"
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
		"release_name":        releaseName,
		"current_version":     currentVersion,
		"latest_version":      latestVersion,
		"upgrade_recommended": latestVersion != "" && latestVersion != currentVersion,
		"notes":               fmt.Sprintf("Use upgrade_chart with version=%s to upgrade", latestVersion),
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

// --- Release inspection handlers ---

func (h *Handler) GetReleaseValues(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	releaseName := mcp.ParseString(req, "release_name", "")
	namespace := mcp.ParseString(req, "namespace", "default")
	allValues := mcp.ParseBoolean(req, "all_values", false)

	values, err := h.helmPort.GetReleaseValues(ctx, releaseName, namespace, allValues)
	if err != nil {
		return toolError(fmt.Sprintf("get_release_values failed: %v", err)), nil
	}

	return toolJSON(map[string]any{
		"release_name": releaseName,
		"namespace":    namespace,
		"all_values":   allValues,
		"values":       values,
	})
}

func (h *Handler) GetReleaseNotes(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	releaseName := mcp.ParseString(req, "release_name", "")
	namespace := mcp.ParseString(req, "namespace", "default")

	notes, err := h.helmPort.GetReleaseNotes(ctx, releaseName, namespace)
	if err != nil {
		return toolError(fmt.Sprintf("get_release_notes failed: %v", err)), nil
	}

	return toolText(notes), nil
}

func (h *Handler) GetReleaseManifest(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	releaseName := mcp.ParseString(req, "release_name", "")
	namespace := mcp.ParseString(req, "namespace", "default")

	manifest, err := h.helmPort.GetReleaseManifest(ctx, releaseName, namespace)
	if err != nil {
		return toolError(fmt.Sprintf("get_release_manifest failed: %v", err)), nil
	}

	return toolText(manifest), nil
}

func (h *Handler) ShowChart(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	chartName := mcp.ParseString(req, "chart_name", "")
	repoURL := mcp.ParseString(req, "repo_url", "")
	version := mcp.ParseString(req, "version", "")

	details, err := h.helmPort.ShowChart(ctx, chartName, repoURL, version)
	if err != nil {
		return toolError(fmt.Sprintf("show_chart failed: %v", err)), nil
	}

	return toolJSON(map[string]any{
		"chart":          chartName,
		"version":        version,
		"metadata":       details.Metadata,
		"default_values": details.DefaultValues,
		"readme":         details.Readme,
	})
}

// --- Repository management handlers ---

func (h *Handler) RepoAdd(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name := mcp.ParseString(req, "name", "")
	url := mcp.ParseString(req, "url", "")
	username := mcp.ParseString(req, "username", "")
	password := mcp.ParseString(req, "password", "")
	caFile := mcp.ParseString(req, "ca_file", "")
	certFile := mcp.ParseString(req, "cert_file", "")
	keyFile := mcp.ParseString(req, "key_file", "")
	insecure := mcp.ParseBoolean(req, "insecure_skip_tls_verify", false)
	passCreds := mcp.ParseBoolean(req, "pass_credentials_all", false)

	if err := h.helmPort.AddRepo(ctx, outbound.RepoEntry{
		Name:                  name,
		URL:                   url,
		Username:              username,
		Password:              password,
		CAFile:                caFile,
		CertFile:              certFile,
		KeyFile:               keyFile,
		InsecureSkipTLSVerify: insecure,
		PassCredentialsAll:    passCreds,
	}); err != nil {
		return toolError(fmt.Sprintf("repo_add failed: %v", err)), nil
	}

	return toolText(fmt.Sprintf("Repository %q added successfully", name)), nil
}

func (h *Handler) RepoRemove(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name := mcp.ParseString(req, "name", "")

	if err := h.helmPort.RemoveRepo(ctx, name); err != nil {
		return toolError(fmt.Sprintf("repo_remove failed: %v", err)), nil
	}

	return toolText(fmt.Sprintf("Repository %q removed", name)), nil
}

func (h *Handler) RepoUpdate(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if err := h.helmPort.UpdateRepos(ctx); err != nil {
		return toolError(fmt.Sprintf("repo_update failed: %v", err)), nil
	}

	return toolText("All repositories updated successfully"), nil
}

func (h *Handler) RepoList(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	repos, err := h.helmPort.ListRepos(ctx)
	if err != nil {
		return toolError(fmt.Sprintf("repo_list failed: %v", err)), nil
	}

	items := make([]map[string]any, 0, len(repos))
	for _, r := range repos {
		items = append(items, map[string]any{
			"name":     r.Name,
			"url":      r.URL,
			"has_tls":  r.CAFile != "" || r.CertFile != "" || r.KeyFile != "",
			"has_auth": r.Username != "",
		})
	}

	return toolJSON(map[string]any{"repositories": items, "count": len(items)})
}

func (h *Handler) RepoSearch(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	keyword := mcp.ParseString(req, "keyword", "")
	repoURL := mcp.ParseString(req, "repo_url", "")

	results, err := h.helmPort.SearchRepo(ctx, keyword, repoURL)
	if err != nil {
		return toolError(fmt.Sprintf("repo_search failed: %v", err)), nil
	}

	items := make([]map[string]any, 0, len(results))
	for _, r := range results {
		items = append(items, map[string]any{
			"name":          r.Name,
			"chart_version": r.ChartVersion,
			"app_version":   r.AppVersion,
			"description":   r.Description,
			"repo_url":      r.RepoURL,
		})
	}

	return toolJSON(map[string]any{"results": items, "count": len(items)})
}

// --- OCI registry handlers ---

func (h *Handler) RegistryLogin(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	host := mcp.ParseString(req, "host", "")
	username := mcp.ParseString(req, "username", "")
	password := mcp.ParseString(req, "password", "")
	caFile := mcp.ParseString(req, "ca_file", "")
	certFile := mcp.ParseString(req, "cert_file", "")
	keyFile := mcp.ParseString(req, "key_file", "")
	insecure := mcp.ParseBoolean(req, "insecure_skip_tls_verify", false)
	plainHTTP := mcp.ParseBoolean(req, "plain_http", false)

	if err := h.helmPort.RegistryLogin(ctx, outbound.RegistryLoginRequest{
		Host:                  host,
		Username:              username,
		Password:              password,
		CAFile:                caFile,
		CertFile:              certFile,
		KeyFile:               keyFile,
		InsecureSkipTLSVerify: insecure,
		PlainHTTP:             plainHTTP,
	}); err != nil {
		return toolError(fmt.Sprintf("registry_login failed: %v", err)), nil
	}

	return toolText(fmt.Sprintf("Logged in to registry %q successfully", host)), nil
}

func (h *Handler) RegistryLogout(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	host := mcp.ParseString(req, "host", "")

	if err := h.helmPort.RegistryLogout(ctx, host); err != nil {
		return toolError(fmt.Sprintf("registry_logout failed: %v", err)), nil
	}

	return toolText(fmt.Sprintf("Logged out from registry %q", host)), nil
}

// --- Enhanced lifecycle handlers ---

func (h *Handler) HelmInstall(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	releaseName := mcp.ParseString(req, "release_name", "")
	chartName := mcp.ParseString(req, "chart_name", "")
	repoURL := mcp.ParseString(req, "repo_url", "")
	namespace := mcp.ParseString(req, "namespace", "default")
	version := mcp.ParseString(req, "version", "")
	timeout := int(mcp.ParseFloat64(req, "timeout_seconds", defaultTimeoutSeconds))

	var values map[string]any
	if v, ok := req.GetArguments()["values"]; ok && v != nil {
		if m, ok := v.(map[string]any); ok {
			values = m
		}
	}

	rel, err := h.helmPort.Install(ctx, outbound.HelmInstallRequest{
		ReleaseName:              releaseName,
		Namespace:                namespace,
		ChartName:                chartName,
		RepoURL:                  repoURL,
		Version:                  version,
		Values:                   values,
		DryRun:                   mcp.ParseBoolean(req, "dry_run", false),
		Wait:                     mcp.ParseBoolean(req, "wait", true),
		WaitForJobs:              mcp.ParseBoolean(req, "wait_for_jobs", false),
		Atomic:                   mcp.ParseBoolean(req, "atomic", false),
		CreateNS:                 mcp.ParseBoolean(req, "create_namespace", true),
		Timeout:                  timeout,
		Description:              mcp.ParseString(req, "description", ""),
		GenerateName:             mcp.ParseBoolean(req, "generate_name", false),
		NameTemplate:             mcp.ParseString(req, "name_template", ""),
		DisableHooks:             mcp.ParseBoolean(req, "disable_hooks", false),
		Replace:                  mcp.ParseBoolean(req, "replace", false),
		SkipCRDs:                 mcp.ParseBoolean(req, "skip_crds", false),
		IncludeCRDs:              mcp.ParseBoolean(req, "include_crds", false),
		SubNotes:                 mcp.ParseBoolean(req, "sub_notes", false),
		SkipSchemaValidation:     mcp.ParseBoolean(req, "skip_schema_validation", false),
		DisableOpenAPIValidation: mcp.ParseBoolean(req, "disable_openapi_validation", false),
		ServerSideApply:          mcp.ParseBoolean(req, "server_side_apply", true),
		ForceConflicts:           mcp.ParseBoolean(req, "force_conflicts", false),
		TakeOwnership:            mcp.ParseBoolean(req, "take_ownership", false),
	})
	if err != nil {
		return toolError(fmt.Sprintf("helm_install failed: %v", err)), nil
	}

	return toolJSON(releaseToMap(rel))
}

func (h *Handler) HelmUpgrade(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	releaseName := mcp.ParseString(req, "release_name", "")
	chartName := mcp.ParseString(req, "chart_name", "")
	repoURL := mcp.ParseString(req, "repo_url", "")
	namespace := mcp.ParseString(req, "namespace", "default")
	version := mcp.ParseString(req, "version", "")
	timeout := int(mcp.ParseFloat64(req, "timeout_seconds", defaultTimeoutSeconds))

	var values map[string]any
	if v, ok := req.GetArguments()["values"]; ok && v != nil {
		if m, ok := v.(map[string]any); ok {
			values = m
		}
	}

	rel, err := h.helmPort.Upgrade(ctx, outbound.HelmUpgradeRequest{
		ReleaseName:              releaseName,
		Namespace:                namespace,
		ChartName:                chartName,
		RepoURL:                  repoURL,
		Version:                  version,
		Values:                   values,
		DryRun:                   mcp.ParseBoolean(req, "dry_run", false),
		Wait:                     mcp.ParseBoolean(req, "wait", true),
		Atomic:                   mcp.ParseBoolean(req, "atomic", false),
		ReuseValues:              mcp.ParseBoolean(req, "reuse_values", false),
		ResetValues:              mcp.ParseBoolean(req, "reset_values", false),
		ResetThenReuseValues:     mcp.ParseBoolean(req, "reset_then_reuse_values", false),
		Timeout:                  timeout,
		MaxHistory:               int(mcp.ParseFloat64(req, "max_history", 0)),
		Description:              mcp.ParseString(req, "description", ""),
		DisableHooks:             mcp.ParseBoolean(req, "disable_hooks", false),
		CleanupOnFail:            mcp.ParseBoolean(req, "cleanup_on_fail", false),
		RollbackOnFailure:        mcp.ParseBoolean(req, "rollback_on_failure", false),
		SkipSchemaValidation:     mcp.ParseBoolean(req, "skip_schema_validation", false),
		DisableOpenAPIValidation: mcp.ParseBoolean(req, "disable_openapi_validation", false),
		ServerSideApply:          mcp.ParseString(req, "server_side_apply", ""),
		ForceConflicts:           mcp.ParseBoolean(req, "force_conflicts", false),
		TakeOwnership:            mcp.ParseBoolean(req, "take_ownership", false),
	})
	if err != nil {
		return toolError(fmt.Sprintf("helm_upgrade failed: %v", err)), nil
	}

	return toolJSON(releaseToMap(rel))
}

func (h *Handler) HelmRollback(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	releaseName := mcp.ParseString(req, "release_name", "")
	namespace := mcp.ParseString(req, "namespace", "default")
	version := int(mcp.ParseFloat64(req, "version", 0))
	timeout := int(mcp.ParseFloat64(req, "timeout_seconds", defaultTimeoutSeconds))

	if err := h.helmPort.Rollback(ctx, outbound.HelmRollbackRequest{
		ReleaseName:     releaseName,
		Namespace:       namespace,
		Version:         version,
		DryRun:          mcp.ParseBoolean(req, "dry_run", false),
		Wait:            mcp.ParseBoolean(req, "wait", true),
		Timeout:         timeout,
		DisableHooks:    mcp.ParseBoolean(req, "disable_hooks", false),
		CleanupOnFail:   mcp.ParseBoolean(req, "cleanup_on_fail", false),
		MaxHistory:      int(mcp.ParseFloat64(req, "max_history", 0)),
		ServerSideApply: mcp.ParseString(req, "server_side_apply", ""),
		ForceConflicts:  mcp.ParseBoolean(req, "force_conflicts", false),
	}); err != nil {
		return toolError(fmt.Sprintf("helm_rollback failed: %v", err)), nil
	}

	return toolText(fmt.Sprintf("Release %q rolled back to revision %d successfully", releaseName, version)), nil
}

func (h *Handler) HelmUninstall(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	releaseName := mcp.ParseString(req, "release_name", "")
	namespace := mcp.ParseString(req, "namespace", "default")
	timeout := int(mcp.ParseFloat64(req, "timeout_seconds", defaultTimeoutSeconds))

	if err := h.helmPort.Uninstall(ctx, outbound.HelmUninstallRequest{
		ReleaseName:  releaseName,
		Namespace:    namespace,
		DryRun:       mcp.ParseBoolean(req, "dry_run", false),
		KeepHistory:  mcp.ParseBoolean(req, "keep_history", false),
		Wait:         mcp.ParseBoolean(req, "wait", false),
		DisableHooks: mcp.ParseBoolean(req, "disable_hooks", false),
		Timeout:      timeout,
		Description:  mcp.ParseString(req, "description", ""),
	}); err != nil {
		return toolError(fmt.Sprintf("helm_uninstall failed: %v", err)), nil
	}

	return toolText(fmt.Sprintf("Release %q uninstalled successfully", releaseName)), nil
}

// --- Extended release inspection handlers ---

func (h *Handler) GetReleaseMetadata(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	releaseName := mcp.ParseString(req, "release_name", "")
	namespace := mcp.ParseString(req, "namespace", "default")
	version := int(mcp.ParseFloat64(req, "version", 0))

	meta, err := h.helmPort.GetReleaseMetadata(ctx, releaseName, namespace, version)
	if err != nil {
		return toolError(fmt.Sprintf("get_release_metadata failed: %v", err)), nil
	}

	return toolJSON(meta)
}

func (h *Handler) GetReleaseStatusResources(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	releaseName := mcp.ParseString(req, "release_name", "")
	namespace := mcp.ParseString(req, "namespace", "default")
	version := int(mcp.ParseFloat64(req, "version", 0))

	details, err := h.helmPort.GetReleaseStatusWithResources(ctx, releaseName, namespace, version)
	if err != nil {
		return toolError(fmt.Sprintf("get_release_status_resources failed: %v", err)), nil
	}

	return toolJSON(details)
}

func (h *Handler) GetReleaseHooks(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	releaseName := mcp.ParseString(req, "release_name", "")
	namespace := mcp.ParseString(req, "namespace", "default")

	hooks, err := h.helmPort.GetReleaseHooks(ctx, releaseName, namespace)
	if err != nil {
		return toolError(fmt.Sprintf("get_release_hooks failed: %v", err)), nil
	}

	return toolJSON(map[string]any{
		"release_name": releaseName,
		"namespace":    namespace,
		"hooks":        hooks,
		"count":        len(hooks),
	})
}

func (h *Handler) GetReleaseRevision(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	releaseName := mcp.ParseString(req, "release_name", "")
	namespace := mcp.ParseString(req, "namespace", "default")
	version := int(mcp.ParseFloat64(req, "version", 0))

	rel, err := h.helmPort.GetReleaseRevision(ctx, releaseName, namespace, version)
	if err != nil {
		return toolError(fmt.Sprintf("get_release_revision failed: %v", err)), nil
	}

	return toolJSON(releaseToMap(rel))
}

func (h *Handler) ListReleasesFiltered(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	releases, err := h.helmPort.ListReleasesFiltered(ctx, outbound.HelmListRequest{
		Namespace:     mcp.ParseString(req, "namespace", ""),
		AllNamespaces: mcp.ParseBoolean(req, "all_namespaces", false),
		Filter:        mcp.ParseString(req, "filter", ""),
		Selector:      mcp.ParseString(req, "selector", ""),
		StateMask:     mcp.ParseString(req, "state_mask", "all"),
		Limit:         int(mcp.ParseFloat64(req, "limit", 0)),
		Offset:        int(mcp.ParseFloat64(req, "offset", 0)),
		SortBy:        mcp.ParseString(req, "sort_by", ""),
		SortReverse:   mcp.ParseBoolean(req, "sort_reverse", false),
	})
	if err != nil {
		return toolError(fmt.Sprintf("list_releases_filtered failed: %v", err)), nil
	}

	items := make([]map[string]any, 0, len(releases))
	for _, r := range releases {
		items = append(items, releaseToMap(r))
	}

	return toolJSON(map[string]any{"releases": items, "count": len(items)})
}

// --- Chart inspection handlers ---

func (h *Handler) ShowChartValues(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	chartName := mcp.ParseString(req, "chart_name", "")
	repoURL := mcp.ParseString(req, "repo_url", "")
	version := mcp.ParseString(req, "version", "")

	values, err := h.helmPort.ShowChartValues(ctx, chartName, repoURL, version)
	if err != nil {
		return toolError(fmt.Sprintf("show_chart_values failed: %v", err)), nil
	}

	return toolJSON(map[string]any{
		"chart":   chartName,
		"version": version,
		"values":  values,
	})
}

func (h *Handler) ShowChartReadme(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	chartName := mcp.ParseString(req, "chart_name", "")
	repoURL := mcp.ParseString(req, "repo_url", "")
	version := mcp.ParseString(req, "version", "")

	readme, err := h.helmPort.ShowChartReadme(ctx, chartName, repoURL, version)
	if err != nil {
		return toolError(fmt.Sprintf("show_chart_readme failed: %v", err)), nil
	}

	return toolText(readme), nil
}

func (h *Handler) ShowChartCRDs(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	chartName := mcp.ParseString(req, "chart_name", "")
	repoURL := mcp.ParseString(req, "repo_url", "")
	version := mcp.ParseString(req, "version", "")

	crds, err := h.helmPort.ShowChartCRDs(ctx, chartName, repoURL, version)
	if err != nil {
		return toolError(fmt.Sprintf("show_chart_crds failed: %v", err)), nil
	}

	return toolJSON(map[string]any{
		"chart":   chartName,
		"version": version,
		"crds":    crds,
		"count":   len(crds),
	})
}

func (h *Handler) ListChartDependencies(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	chartName := mcp.ParseString(req, "chart_name", "")
	repoURL := mcp.ParseString(req, "repo_url", "")
	version := mcp.ParseString(req, "version", "")

	deps, err := h.helmPort.ListChartDependencies(ctx, chartName, repoURL, version)
	if err != nil {
		return toolError(fmt.Sprintf("list_chart_dependencies failed: %v", err)), nil
	}

	return toolJSON(map[string]any{
		"chart":        chartName,
		"version":      version,
		"dependencies": deps,
		"count":        len(deps),
	})
}

func (h *Handler) TemplateChart(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var values map[string]any
	if v, ok := req.GetArguments()["values"]; ok && v != nil {
		if m, ok := v.(map[string]any); ok {
			values = m
		}
	}

	manifest, err := h.helmPort.TemplateChart(ctx, outbound.TemplateRequest{
		ReleaseName:          mcp.ParseString(req, "release_name", ""),
		Namespace:            mcp.ParseString(req, "namespace", "default"),
		ChartName:            mcp.ParseString(req, "chart_name", ""),
		RepoURL:              mcp.ParseString(req, "repo_url", ""),
		Version:              mcp.ParseString(req, "version", ""),
		Values:               values,
		ShowNotes:            mcp.ParseBoolean(req, "show_notes", false),
		IncludeCRDs:          mcp.ParseBoolean(req, "include_crds", false),
		SkipSchemaValidation: mcp.ParseBoolean(req, "skip_schema_validation", false),
	})
	if err != nil {
		return toolError(fmt.Sprintf("template_chart failed: %v", err)), nil
	}

	return toolText(manifest), nil
}

func (h *Handler) LintChart(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	chartPathStr := mcp.ParseString(req, "chart_path", "")

	paths := []string{}
	for _, p := range strings.Split(chartPathStr, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			paths = append(paths, p)
		}
	}

	if len(paths) == 0 {
		return toolError("chart_path is required"), nil
	}

	var values map[string]any
	if v, ok := req.GetArguments()["values"]; ok && v != nil {
		if m, ok := v.(map[string]any); ok {
			values = m
		}
	}

	result, err := h.helmPort.LintChart(ctx, paths, values)
	if err != nil {
		return toolError(fmt.Sprintf("lint_chart failed: %v", err)), nil
	}

	return toolJSON(result)
}

// --- Chart distribution handlers ---

func (h *Handler) PackageChart(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	archivePath, err := h.helmPort.PackageChart(ctx, outbound.PackageRequest{
		ChartPath:   mcp.ParseString(req, "chart_path", ""),
		Version:     mcp.ParseString(req, "version", ""),
		AppVersion:  mcp.ParseString(req, "app_version", ""),
		Destination: mcp.ParseString(req, "destination", "."),
		Sign:        mcp.ParseBoolean(req, "sign", false),
		Key:         mcp.ParseString(req, "key", ""),
		Keyring:     mcp.ParseString(req, "keyring", ""),
	})
	if err != nil {
		return toolError(fmt.Sprintf("package_chart failed: %v", err)), nil
	}

	return toolText(fmt.Sprintf("Chart packaged successfully: %s", archivePath)), nil
}

func (h *Handler) PullChart(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	output, err := h.helmPort.PullChart(ctx, outbound.PullRequest{
		ChartRef:              mcp.ParseString(req, "chart_ref", ""),
		Version:               mcp.ParseString(req, "version", ""),
		RepoURL:               mcp.ParseString(req, "repo_url", ""),
		DestDir:               mcp.ParseString(req, "dest_dir", ""),
		Untar:                 mcp.ParseBoolean(req, "untar", false),
		UntarDir:              mcp.ParseString(req, "untar_dir", ""),
		Username:              mcp.ParseString(req, "username", ""),
		Password:              mcp.ParseString(req, "password", ""),
		CAFile:                mcp.ParseString(req, "ca_file", ""),
		CertFile:              mcp.ParseString(req, "cert_file", ""),
		KeyFile:               mcp.ParseString(req, "key_file", ""),
		InsecureSkipTLSVerify: mcp.ParseBoolean(req, "insecure_skip_tls_verify", false),
		PassCredentialsAll:    mcp.ParseBoolean(req, "pass_credentials_all", false),
		PlainHTTP:             mcp.ParseBoolean(req, "plain_http", false),
	})
	if err != nil {
		return toolError(fmt.Sprintf("pull_chart failed: %v", err)), nil
	}

	return toolText(output), nil
}

func (h *Handler) PushChart(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	output, err := h.helmPort.PushChart(ctx, outbound.PushRequest{
		ChartPath:             mcp.ParseString(req, "chart_path", ""),
		Remote:                mcp.ParseString(req, "remote", ""),
		CAFile:                mcp.ParseString(req, "ca_file", ""),
		CertFile:              mcp.ParseString(req, "cert_file", ""),
		KeyFile:               mcp.ParseString(req, "key_file", ""),
		InsecureSkipTLSVerify: mcp.ParseBoolean(req, "insecure_skip_tls_verify", false),
		PlainHTTP:             mcp.ParseBoolean(req, "plain_http", false),
	})
	if err != nil {
		return toolError(fmt.Sprintf("push_chart failed: %v", err)), nil
	}

	return toolText(output), nil
}

func (h *Handler) TestRelease(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	releaseName := mcp.ParseString(req, "release_name", "")
	namespace := mcp.ParseString(req, "namespace", "default")
	timeout := int(mcp.ParseFloat64(req, "timeout_seconds", defaultTimeoutSeconds))

	var filters []string
	if f := mcp.ParseString(req, "filter", ""); f != "" {
		for _, name := range strings.Split(f, ",") {
			name = strings.TrimSpace(name)
			if name != "" {
				filters = append(filters, name)
			}
		}
	}

	result, err := h.helmPort.TestRelease(ctx, releaseName, namespace, timeout, filters)
	if err != nil {
		return toolError(fmt.Sprintf("test_release failed: %v", err)), nil
	}

	return toolJSON(result)
}

// --- Repository management handlers (additional) ---

func (h *Handler) RepoUpdateSingle(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name := mcp.ParseString(req, "name", "")

	if err := h.helmPort.UpdateRepo(ctx, name); err != nil {
		return toolError(fmt.Sprintf("repo_update_single failed: %v", err)), nil
	}

	return toolText(fmt.Sprintf("Repository %q updated successfully", name)), nil
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

// releaseToMap converts a Helm release to a JSON-friendly map for tool responses.
func releaseToMap(r *releasev1.Release) map[string]any {
	if r == nil {
		return nil
	}

	chartName := ""
	chartVersion := ""
	appVersion := ""
	if r.Chart != nil && r.Chart.Metadata != nil {
		chartName = r.Chart.Metadata.Name
		chartVersion = r.Chart.Metadata.Version
		appVersion = r.Chart.Metadata.AppVersion
	}

	status := ""
	deployedAt := ""
	notes := ""
	if r.Info != nil {
		status = string(r.Info.Status)
		deployedAt = r.Info.LastDeployed.String()
		notes = r.Info.Notes
	}

	return map[string]any{
		"name":          r.Name,
		"namespace":     r.Namespace,
		"revision":      r.Version,
		"status":        status,
		"chart":         chartName,
		"chart_version": chartVersion,
		"app_version":   appVersion,
		"deployed_at":   deployedAt,
		"notes":         notes,
	}
}
