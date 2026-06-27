package mcp

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	appcluster "github.com/vinaycharlie01/sh-mcp-go/internal/application/cluster"
	appdeployment "github.com/vinaycharlie01/sh-mcp-go/internal/application/deployment"
	appplanner "github.com/vinaycharlie01/sh-mcp-go/internal/application/planner"
	"github.com/vinaycharlie01/sh-mcp-go/internal/infrastructure/config"
	"github.com/vinaycharlie01/sh-mcp-go/internal/ports/outbound"
)

// Server wraps the MCP server and registers all sh-mcp-go tools.
type Server struct {
	mcp     *server.MCPServer
	handler *Handler
	cfg     *config.MCPConfig
}

// NewServer creates and configures the MCP server with all tools registered.
func NewServer(
	cfg *config.MCPConfig,
	deploymentSvc *appdeployment.Service,
	clusterSvc *appcluster.Service,
	plannerSvc *appplanner.Service,
	helmPort outbound.HelmPort,
) *Server {
	s := server.NewMCPServer(
		cfg.Name,
		cfg.Version,
		server.WithToolCapabilities(true),
		server.WithRecovery(),
	)

	h := &Handler{
		deploymentSvc: deploymentSvc,
		clusterSvc:    clusterSvc,
		plannerSvc:    plannerSvc,
		helmPort:      helmPort,
	}

	registerTools(s, h)

	return &Server{
		mcp:     s,
		handler: h,
		cfg:     cfg,
	}
}

// ServeStdio runs the MCP server over stdin/stdout (standard MCP transport).
func (s *Server) ServeStdio(_ context.Context) error {
	slog.Info("starting MCP server (stdio transport)")

	return server.ServeStdio(s.mcp)
}

// ServeSSE runs the MCP server as an SSE HTTP endpoint.
func (s *Server) ServeSSE(_ context.Context) error {
	addr := s.cfg.SSEAddr
	if addr == "" {
		addr = "0.0.0.0:8081"
	}
	slog.Info("starting MCP server (SSE transport)", slog.String("addr", addr))
	sseServer := server.NewSSEServer(s.mcp, server.WithBaseURL(fmt.Sprintf("http://%s", addr)))

	return sseServer.Start(addr)
}

// registerTools registers all 55 MCP tools onto the server.
func registerTools(s *server.MCPServer, h *Handler) {
	// ---- Chart lifecycle ----
	s.AddTool(toolInstallChart(), h.InstallChart)
	s.AddTool(toolUpgradeChart(), h.UpgradeChart)
	s.AddTool(toolRollbackChart(), h.RollbackChart)
	s.AddTool(toolUninstallChart(), h.UninstallChart)

	// ---- Enhanced chart lifecycle (full Helm v4 params) ----
	s.AddTool(toolHelmInstall(), h.HelmInstall)
	s.AddTool(toolHelmUpgrade(), h.HelmUpgrade)
	s.AddTool(toolHelmRollback(), h.HelmRollback)
	s.AddTool(toolHelmUninstall(), h.HelmUninstall)

	// ---- Operator lifecycle ----
	s.AddTool(toolInstallOperator(), h.InstallOperator)
	s.AddTool(toolUpgradeOperator(), h.UpgradeOperator)
	s.AddTool(toolRollbackOperator(), h.RollbackOperator)
	s.AddTool(toolDeleteOperator(), h.DeleteOperator)

	// ---- Planning & validation ----
	s.AddTool(toolPlanDeployment(), h.PlanDeployment)
	s.AddTool(toolValidateCluster(), h.ValidateCluster)
	s.AddTool(toolValidateRelease(), h.ValidateRelease)

	// ---- Inventory ----
	s.AddTool(toolClusterInventory(), h.ClusterInventory)
	s.AddTool(toolReleaseInventory(), h.ReleaseInventory)

	// ---- Analysis ----
	s.AddTool(toolResourceEstimation(), h.ResourceEstimation)
	s.AddTool(toolDependencyAnalysis(), h.DependencyAnalysis)
	s.AddTool(toolSecurityScan(), h.SecurityScan)
	s.AddTool(toolHealthCheck(), h.HealthCheck)
	s.AddTool(toolGenerateRCA(), h.GenerateRCA)
	s.AddTool(toolAnalyzeFailure(), h.AnalyzeFailure)

	// ---- Recommendations ----
	s.AddTool(toolGenerateValuesYAML(), h.GenerateValuesYAML)
	s.AddTool(toolRecommendUpgrade(), h.RecommendUpgrade)
	s.AddTool(toolRecommendOperator(), h.RecommendOperator)

	// ---- Status ----
	s.AddTool(toolDeploymentStatus(), h.DeploymentStatus)
	s.AddTool(toolReleaseStatus(), h.ReleaseStatus)

	// ---- Release inspection ----
	s.AddTool(toolGetReleaseValues(), h.GetReleaseValues)
	s.AddTool(toolGetReleaseNotes(), h.GetReleaseNotes)
	s.AddTool(toolGetReleaseManifest(), h.GetReleaseManifest)
	s.AddTool(toolShowChart(), h.ShowChart)

	// ---- Extended release inspection ----
	s.AddTool(toolGetReleaseMetadata(), h.GetReleaseMetadata)
	s.AddTool(toolGetReleaseStatusResources(), h.GetReleaseStatusResources)
	s.AddTool(toolGetReleaseHooks(), h.GetReleaseHooks)
	s.AddTool(toolGetReleaseRevision(), h.GetReleaseRevision)
	s.AddTool(toolListReleasesFiltered(), h.ListReleasesFiltered)

	// ---- Chart inspection ----
	s.AddTool(toolShowChartValues(), h.ShowChartValues)
	s.AddTool(toolShowChartReadme(), h.ShowChartReadme)
	s.AddTool(toolShowChartCRDs(), h.ShowChartCRDs)
	s.AddTool(toolListChartDependencies(), h.ListChartDependencies)
	s.AddTool(toolTemplateChart(), h.TemplateChart)
	s.AddTool(toolLintChart(), h.LintChart)

	// ---- Chart distribution ----
	s.AddTool(toolPackageChart(), h.PackageChart)
	s.AddTool(toolPullChart(), h.PullChart)
	s.AddTool(toolPushChart(), h.PushChart)
	s.AddTool(toolTestRelease(), h.TestRelease)

	// ---- Repository management ----
	s.AddTool(toolRepoAdd(), h.RepoAdd)
	s.AddTool(toolRepoRemove(), h.RepoRemove)
	s.AddTool(toolRepoUpdate(), h.RepoUpdate)
	s.AddTool(toolRepoUpdateSingle(), h.RepoUpdateSingle)
	s.AddTool(toolRepoList(), h.RepoList)
	s.AddTool(toolRepoSearch(), h.RepoSearch)

	// ---- OCI registry ----
	s.AddTool(toolRegistryLogin(), h.RegistryLogin)
	s.AddTool(toolRegistryLogout(), h.RegistryLogout)
}

// --- Tool definitions ---

func toolInstallChart() mcp.Tool {
	return mcp.NewTool("install_chart",
		mcp.WithDescription("Install a Helm chart onto a Kubernetes cluster. Supports all major chart repositories."),
		mcp.WithString("release_name", mcp.Required(), mcp.Description("Helm release name (lowercase alphanumeric, hyphens allowed)")),
		mcp.WithString("chart_name", mcp.Required(), mcp.Description("Chart name as it appears in the repository")),
		mcp.WithString("repo_url", mcp.Required(), mcp.Description("Helm repository URL")),
		mcp.WithString("namespace", mcp.Description("Target Kubernetes namespace (default: default)")),
		mcp.WithString("version", mcp.Description("Chart version (omit for latest)")),
		mcp.WithObject("values", mcp.Description("Values overrides (equivalent to -f values.yaml)")),
		mcp.WithBoolean("dry_run", mcp.Description("Perform a dry run without applying changes")),
		mcp.WithBoolean("wait", mcp.Description("Wait for resources to become ready")),
		mcp.WithBoolean("atomic", mcp.Description("Roll back on failure")),
		mcp.WithBoolean("create_namespace", mcp.Description("Create namespace if it doesn't exist")),
		mcp.WithNumber("timeout_seconds", mcp.Description("Operation timeout in seconds (default: 300)")),
	)
}

func toolUpgradeChart() mcp.Tool {
	return mcp.NewTool("upgrade_chart",
		mcp.WithDescription("Upgrade an existing Helm release to a new chart version or with new values."),
		mcp.WithString("release_name", mcp.Required(), mcp.Description("Existing release name")),
		mcp.WithString("chart_name", mcp.Required(), mcp.Description("Chart name")),
		mcp.WithString("repo_url", mcp.Required(), mcp.Description("Helm repository URL")),
		mcp.WithString("namespace", mcp.Description("Release namespace")),
		mcp.WithString("version", mcp.Description("Target chart version")),
		mcp.WithObject("values", mcp.Description("Values overrides")),
		mcp.WithBoolean("dry_run", mcp.Description("Perform a dry run")),
		mcp.WithBoolean("wait", mcp.Description("Wait for readiness")),
		mcp.WithBoolean("atomic", mcp.Description("Roll back on failure")),
		mcp.WithBoolean("reuse_values", mcp.Description("Reuse existing release values")),
		mcp.WithBoolean("reset_values", mcp.Description("Reset values to chart defaults")),
	)
}

func toolRollbackChart() mcp.Tool {
	return mcp.NewTool("rollback_chart",
		mcp.WithDescription("Roll back a Helm release to a previous revision."),
		mcp.WithString("release_name", mcp.Required(), mcp.Description("Release name to roll back")),
		mcp.WithString("namespace", mcp.Required(), mcp.Description("Release namespace")),
		mcp.WithNumber("version", mcp.Description("Target revision number (0 = previous)")),
		mcp.WithBoolean("dry_run", mcp.Description("Perform a dry run")),
		mcp.WithBoolean("wait", mcp.Description("Wait for readiness after rollback")),
	)
}

func toolUninstallChart() mcp.Tool {
	return mcp.NewTool("uninstall_chart",
		mcp.WithDescription("Uninstall a Helm release from the cluster."),
		mcp.WithString("release_name", mcp.Required(), mcp.Description("Release name to uninstall")),
		mcp.WithString("namespace", mcp.Required(), mcp.Description("Release namespace")),
		mcp.WithBoolean("dry_run", mcp.Description("Perform a dry run")),
		mcp.WithBoolean("keep_history", mcp.Description("Retain release history")),
	)
}

func toolInstallOperator() mcp.Tool {
	return mcp.NewTool("install_operator",
		mcp.WithDescription("Install a Kubernetes operator via OLM or Helm-based operator chart."),
		mcp.WithString("name", mcp.Required(), mcp.Description("Operator name")),
		mcp.WithString("namespace", mcp.Description("Installation namespace")),
		mcp.WithString("channel", mcp.Description("OLM subscription channel")),
		mcp.WithString("source", mcp.Description("Catalog source name")),
	)
}

func toolUpgradeOperator() mcp.Tool {
	return mcp.NewTool("upgrade_operator",
		mcp.WithDescription("Upgrade an installed Kubernetes operator to a new channel or version."),
		mcp.WithString("name", mcp.Required(), mcp.Description("Operator name")),
		mcp.WithString("namespace", mcp.Required(), mcp.Description("Operator namespace")),
		mcp.WithString("channel", mcp.Description("New channel")),
		mcp.WithString("version", mcp.Description("Target version")),
	)
}

func toolRollbackOperator() mcp.Tool {
	return mcp.NewTool("rollback_operator",
		mcp.WithDescription("Roll back an operator upgrade."),
		mcp.WithString("name", mcp.Required(), mcp.Description("Operator name")),
		mcp.WithString("namespace", mcp.Required(), mcp.Description("Operator namespace")),
	)
}

func toolDeleteOperator() mcp.Tool {
	return mcp.NewTool("delete_operator",
		mcp.WithDescription("Delete an installed Kubernetes operator."),
		mcp.WithString("name", mcp.Required(), mcp.Description("Operator name")),
		mcp.WithString("namespace", mcp.Required(), mcp.Description("Operator namespace")),
	)
}

func toolPlanDeployment() mcp.Tool {
	return mcp.NewTool("plan_deployment",
		mcp.WithDescription(
			"Generate an AI deployment plan from natural language intent. Returns an ordered list of steps with dependency graph.",
		),
		mcp.WithString("intent", mcp.Required(), mcp.Description(
			"Natural language deployment intent, e.g. 'Deploy Prometheus and Grafana with persistence'",
		)),
		mcp.WithString("namespace", mcp.Description("Target namespace (default: default)")),
	)
}

func toolValidateCluster() mcp.Tool {
	return mcp.NewTool("validate_cluster",
		mcp.WithDescription("Run prerequisite checks on the cluster to ensure it's ready for deployments."),
	)
}

func toolValidateRelease() mcp.Tool {
	return mcp.NewTool("validate_release",
		mcp.WithDescription("Validate an existing Helm release and its resource health."),
		mcp.WithString("release_name", mcp.Required(), mcp.Description("Release name")),
		mcp.WithString("namespace", mcp.Required(), mcp.Description("Release namespace")),
	)
}

func toolClusterInventory() mcp.Tool {
	return mcp.NewTool("cluster_inventory",
		mcp.WithDescription("Return a complete inventory of the cluster: nodes, namespaces, Helm releases, CRDs."),
	)
}

func toolReleaseInventory() mcp.Tool {
	return mcp.NewTool("release_inventory",
		mcp.WithDescription("List all Helm releases, optionally filtered by namespace."),
		mcp.WithString("namespace", mcp.Description("Filter by namespace (omit for all namespaces)")),
	)
}

func toolResourceEstimation() mcp.Tool {
	return mcp.NewTool("resource_estimation",
		mcp.WithDescription("Estimate CPU, memory, and storage requirements for a given chart deployment."),
		mcp.WithString("chart_name", mcp.Required(), mcp.Description("Chart name")),
		mcp.WithString("namespace", mcp.Description("Target namespace")),
		mcp.WithNumber("replicas", mcp.Description("Number of replicas (default: 1)")),
	)
}

func toolDependencyAnalysis() mcp.Tool {
	return mcp.NewTool("dependency_analysis",
		mcp.WithDescription("Analyse chart dependencies and identify required CRDs, operators, and charts to install first."),
		mcp.WithString("chart_name", mcp.Required(), mcp.Description("Chart name")),
		mcp.WithString("repo_url", mcp.Required(), mcp.Description("Helm repository URL")),
		mcp.WithString("version", mcp.Description("Chart version")),
	)
}

func toolSecurityScan() mcp.Tool {
	return mcp.NewTool("security_scan",
		mcp.WithDescription("Scan a Helm release or chart for security issues: RBAC, privilege escalation, image vulnerabilities."),
		mcp.WithString("release_name", mcp.Description("Release name to scan (or use chart_name for pre-install scan)")),
		mcp.WithString("namespace", mcp.Description("Release namespace")),
		mcp.WithString("chart_name", mcp.Description("Chart name for pre-install scan")),
		mcp.WithString("repo_url", mcp.Description("Repo URL for pre-install scan")),
	)
}

func toolHealthCheck() mcp.Tool {
	return mcp.NewTool("health_check",
		mcp.WithDescription("Check the health of all resources in a Helm release."),
		mcp.WithString("release_name", mcp.Required(), mcp.Description("Release name")),
		mcp.WithString("namespace", mcp.Required(), mcp.Description("Release namespace")),
	)
}

func toolGenerateRCA() mcp.Tool {
	return mcp.NewTool("generate_rca",
		mcp.WithDescription("Generate a root cause analysis for a failing or degraded Helm release."),
		mcp.WithString("release_name", mcp.Required(), mcp.Description("Release name")),
		mcp.WithString("namespace", mcp.Required(), mcp.Description("Release namespace")),
	)
}

func toolAnalyzeFailure() mcp.Tool {
	return mcp.NewTool("analyze_failure",
		mcp.WithDescription("Analyse a deployment failure and suggest remediation steps."),
		mcp.WithString("release_name", mcp.Required(), mcp.Description("Failed release name")),
		mcp.WithString("namespace", mcp.Required(), mcp.Description("Release namespace")),
		mcp.WithString("error_message", mcp.Description("Error message from the failed deployment")),
	)
}

func toolGenerateValuesYAML() mcp.Tool {
	return mcp.NewTool("generate_values_yaml",
		mcp.WithDescription("Generate a values.yaml skeleton for a Helm chart with defaults and annotations."),
		mcp.WithString("chart_name", mcp.Required(), mcp.Description("Chart name")),
		mcp.WithString("repo_url", mcp.Required(), mcp.Description("Repository URL")),
		mcp.WithString("version", mcp.Description("Chart version")),
	)
}

func toolRecommendUpgrade() mcp.Tool {
	return mcp.NewTool("recommend_upgrade",
		mcp.WithDescription("Recommend whether and how to upgrade a Helm release, including version and values changes."),
		mcp.WithString("release_name", mcp.Required(), mcp.Description("Release name")),
		mcp.WithString("namespace", mcp.Required(), mcp.Description("Release namespace")),
	)
}

func toolRecommendOperator() mcp.Tool {
	return mcp.NewTool("recommend_operator",
		mcp.WithDescription("Recommend a Kubernetes operator for a given workload type."),
		mcp.WithString("workload_type", mcp.Required(), mcp.Description("Workload type, e.g. 'database', 'messaging', 'monitoring'")),
	)
}

func toolDeploymentStatus() mcp.Tool {
	return mcp.NewTool("deployment_status",
		mcp.WithDescription("Get the current status of a deployment tracked by sh-mcp-go."),
		mcp.WithString("deployment_id", mcp.Description("Deployment ID")),
		mcp.WithString("release_name", mcp.Description("Release name (alternative to deployment_id)")),
		mcp.WithString("namespace", mcp.Description("Release namespace")),
	)
}

func toolReleaseStatus() mcp.Tool {
	return mcp.NewTool("release_status",
		mcp.WithDescription("Get the current status of a Helm release directly from the cluster."),
		mcp.WithString("release_name", mcp.Required(), mcp.Description("Release name")),
		mcp.WithString("namespace", mcp.Required(), mcp.Description("Release namespace")),
	)
}

func toolGetReleaseValues() mcp.Tool {
	return mcp.NewTool("get_release_values",
		mcp.WithDescription("Retrieve the values used to deploy a Helm release."),
		mcp.WithString("release_name", mcp.Required(), mcp.Description("Release name")),
		mcp.WithString("namespace", mcp.Required(), mcp.Description("Release namespace")),
		mcp.WithBoolean("all_values", mcp.Description("Include chart default values (default: user-supplied only)")),
	)
}

func toolGetReleaseNotes() mcp.Tool {
	return mcp.NewTool("get_release_notes",
		mcp.WithDescription("Retrieve the notes produced by a deployed Helm release."),
		mcp.WithString("release_name", mcp.Required(), mcp.Description("Release name")),
		mcp.WithString("namespace", mcp.Required(), mcp.Description("Release namespace")),
	)
}

func toolGetReleaseManifest() mcp.Tool {
	return mcp.NewTool("get_release_manifest",
		mcp.WithDescription("Retrieve the Kubernetes manifests generated by a Helm release."),
		mcp.WithString("release_name", mcp.Required(), mcp.Description("Release name")),
		mcp.WithString("namespace", mcp.Required(), mcp.Description("Release namespace")),
	)
}

func toolShowChart() mcp.Tool {
	return mcp.NewTool("show_chart",
		mcp.WithDescription("Show metadata, default values and README for a Helm chart without installing it."),
		mcp.WithString("chart_name", mcp.Required(), mcp.Description("Chart name")),
		mcp.WithString("repo_url", mcp.Required(), mcp.Description("Helm repository URL")),
		mcp.WithString("version", mcp.Description("Chart version (omit for latest)")),
	)
}

func toolRepoAdd() mcp.Tool {
	return mcp.NewTool("repo_add",
		mcp.WithDescription("Add a Helm chart repository to the local configuration."),
		mcp.WithString("name", mcp.Required(), mcp.Description("Repository alias name")),
		mcp.WithString("url", mcp.Required(), mcp.Description("Repository URL")),
		mcp.WithString("username", mcp.Description("Basic auth username")),
		mcp.WithString("password", mcp.Description("Basic auth password")),
		mcp.WithString("ca_file", mcp.Description("Path to CA certificate file")),
		mcp.WithString("cert_file", mcp.Description("Path to TLS client certificate file")),
		mcp.WithString("key_file", mcp.Description("Path to TLS client key file")),
		mcp.WithBoolean("insecure_skip_tls_verify", mcp.Description("Disable TLS certificate verification")),
		mcp.WithBoolean("pass_credentials_all", mcp.Description("Pass credentials to all chart download hosts")),
	)
}

func toolRepoRemove() mcp.Tool {
	return mcp.NewTool("repo_remove",
		mcp.WithDescription("Remove a Helm chart repository from local configuration."),
		mcp.WithString("name", mcp.Required(), mcp.Description("Repository alias name to remove")),
	)
}

func toolRepoUpdate() mcp.Tool {
	return mcp.NewTool("repo_update",
		mcp.WithDescription("Refresh the index for all configured Helm chart repositories."),
	)
}

func toolRepoList() mcp.Tool {
	return mcp.NewTool("repo_list",
		mcp.WithDescription("List all configured Helm chart repositories."),
	)
}

func toolRepoSearch() mcp.Tool {
	return mcp.NewTool("repo_search",
		mcp.WithDescription("Search configured Helm repositories for charts matching a keyword."),
		mcp.WithString("keyword", mcp.Description("Search keyword (matches chart name)")),
		mcp.WithString("repo_url", mcp.Description("Limit search to a specific repository URL")),
	)
}

func toolRegistryLogin() mcp.Tool {
	return mcp.NewTool("registry_login",
		mcp.WithDescription("Authenticate with an OCI container registry to pull or push Helm charts."),
		mcp.WithString("host", mcp.Required(), mcp.Description("OCI registry hostname (e.g. registry.example.com)")),
		mcp.WithString("username", mcp.Required(), mcp.Description("Registry username")),
		mcp.WithString("password", mcp.Required(), mcp.Description("Registry password or token")),
		mcp.WithString("ca_file", mcp.Description("Path to CA certificate for registry TLS")),
		mcp.WithString("cert_file", mcp.Description("Path to client TLS certificate")),
		mcp.WithString("key_file", mcp.Description("Path to client TLS key")),
		mcp.WithBoolean("insecure_skip_tls_verify", mcp.Description("Disable TLS certificate verification")),
		mcp.WithBoolean("plain_http", mcp.Description("Use plain HTTP instead of HTTPS")),
	)
}

func toolRegistryLogout() mcp.Tool {
	return mcp.NewTool("registry_logout",
		mcp.WithDescription("Remove stored credentials for an OCI container registry."),
		mcp.WithString("host", mcp.Required(), mcp.Description("OCI registry hostname")),
	)
}

// --- Enhanced lifecycle tool definitions ---

func toolHelmInstall() mcp.Tool {
	return mcp.NewTool("helm_install",
		mcp.WithDescription("Install a Helm chart with full v4 support: server-side apply, ownership takeover, hooks control, CRD management."),
		mcp.WithString("release_name", mcp.Required(), mcp.Description("Helm release name")),
		mcp.WithString("chart_name", mcp.Required(), mcp.Description("Chart name or OCI reference")),
		mcp.WithString("repo_url", mcp.Description("Helm repository URL")),
		mcp.WithString("namespace", mcp.Description("Target namespace (default: default)")),
		mcp.WithString("version", mcp.Description("Chart version (omit for latest)")),
		mcp.WithObject("values", mcp.Description("Values overrides")),
		mcp.WithBoolean("dry_run", mcp.Description("Simulate the install without applying")),
		mcp.WithBoolean("wait", mcp.Description("Wait for resources to become ready")),
		mcp.WithBoolean("wait_for_jobs", mcp.Description("Wait for jobs to complete")),
		mcp.WithBoolean("atomic", mcp.Description("Roll back on failure")),
		mcp.WithBoolean("create_namespace", mcp.Description("Create the namespace if it does not exist")),
		mcp.WithNumber("timeout_seconds", mcp.Description("Operation timeout in seconds")),
		mcp.WithString("description", mcp.Description("Human-readable description for this release")),
		mcp.WithBoolean("generate_name", mcp.Description("Auto-generate a release name")),
		mcp.WithString("name_template", mcp.Description("Go template for release name generation")),
		mcp.WithBoolean("disable_hooks", mcp.Description("Disable pre/post install hooks")),
		mcp.WithBoolean("replace", mcp.Description("Re-use an existing release name")),
		mcp.WithBoolean("skip_crds", mcp.Description("Skip CRD installation")),
		mcp.WithBoolean("include_crds", mcp.Description("Include CRDs in the rendered output")),
		mcp.WithBoolean("sub_notes", mcp.Description("Render sub-chart NOTES.txt")),
		mcp.WithBoolean("skip_schema_validation", mcp.Description("Skip values JSON schema validation")),
		mcp.WithBoolean("disable_openapi_validation", mcp.Description("Disable OpenAPI schema validation")),
		mcp.WithBoolean("server_side_apply", mcp.Description("Use Kubernetes server-side apply")),
		mcp.WithBoolean("force_conflicts", mcp.Description("Force field ownership conflicts during server-side apply")),
		mcp.WithBoolean("take_ownership", mcp.Description("Adopt existing resources not managed by Helm")),
	)
}

func toolHelmUpgrade() mcp.Tool {
	return mcp.NewTool("helm_upgrade",
		mcp.WithDescription("Upgrade a Helm release with full v4 support: history limits, cleanup policies, and server-side apply."),
		mcp.WithString("release_name", mcp.Required(), mcp.Description("Release name")),
		mcp.WithString("chart_name", mcp.Required(), mcp.Description("Chart name")),
		mcp.WithString("repo_url", mcp.Description("Helm repository URL")),
		mcp.WithString("namespace", mcp.Description("Release namespace")),
		mcp.WithString("version", mcp.Description("Target chart version")),
		mcp.WithObject("values", mcp.Description("Values overrides")),
		mcp.WithBoolean("dry_run", mcp.Description("Simulate the upgrade")),
		mcp.WithBoolean("wait", mcp.Description("Wait for resources to become ready")),
		mcp.WithBoolean("atomic", mcp.Description("Roll back on failure")),
		mcp.WithBoolean("reuse_values", mcp.Description("Reuse existing release values")),
		mcp.WithBoolean("reset_values", mcp.Description("Reset values to chart defaults")),
		mcp.WithBoolean("reset_then_reuse_values", mcp.Description("Reset to chart defaults then merge with last supplied values")),
		mcp.WithNumber("timeout_seconds", mcp.Description("Operation timeout in seconds")),
		mcp.WithNumber("max_history", mcp.Description("Maximum number of revision history entries to keep")),
		mcp.WithString("description", mcp.Description("Human-readable description for this upgrade")),
		mcp.WithBoolean("disable_hooks", mcp.Description("Disable pre/post upgrade hooks")),
		mcp.WithBoolean("cleanup_on_fail", mcp.Description("Delete newly created resources on upgrade failure")),
		mcp.WithBoolean("rollback_on_failure", mcp.Description("Automatically roll back on upgrade failure")),
		mcp.WithBoolean("skip_schema_validation", mcp.Description("Skip values JSON schema validation")),
		mcp.WithBoolean("disable_openapi_validation", mcp.Description("Disable OpenAPI schema validation")),
		mcp.WithString("server_side_apply", mcp.Description("Server-side apply mode: auto, true, or false")),
		mcp.WithBoolean("force_conflicts", mcp.Description("Force field ownership conflicts during server-side apply")),
		mcp.WithBoolean("take_ownership", mcp.Description("Adopt existing resources not managed by Helm")),
	)
}

func toolHelmRollback() mcp.Tool {
	return mcp.NewTool("helm_rollback",
		mcp.WithDescription("Roll back a Helm release with full v4 support: hooks control, cleanup policies, and server-side apply."),
		mcp.WithString("release_name", mcp.Required(), mcp.Description("Release name")),
		mcp.WithString("namespace", mcp.Required(), mcp.Description("Release namespace")),
		mcp.WithNumber("version", mcp.Description("Target revision (0 = previous)")),
		mcp.WithBoolean("dry_run", mcp.Description("Simulate the rollback")),
		mcp.WithBoolean("wait", mcp.Description("Wait for resources after rollback")),
		mcp.WithNumber("timeout_seconds", mcp.Description("Operation timeout in seconds")),
		mcp.WithBoolean("disable_hooks", mcp.Description("Disable pre/post rollback hooks")),
		mcp.WithBoolean("cleanup_on_fail", mcp.Description("Delete newly created resources on rollback failure")),
		mcp.WithNumber("max_history", mcp.Description("Maximum number of revision history entries to keep")),
		mcp.WithString("server_side_apply", mcp.Description("Server-side apply mode: auto, true, or false")),
		mcp.WithBoolean("force_conflicts", mcp.Description("Force field ownership conflicts")),
	)
}

func toolHelmUninstall() mcp.Tool {
	return mcp.NewTool("helm_uninstall",
		mcp.WithDescription("Uninstall a Helm release with full Helm v4 parameter support including hooks control and wait options."),
		mcp.WithString("release_name", mcp.Required(), mcp.Description("Release name to uninstall")),
		mcp.WithString("namespace", mcp.Required(), mcp.Description("Release namespace")),
		mcp.WithBoolean("dry_run", mcp.Description("Simulate the uninstall")),
		mcp.WithBoolean("keep_history", mcp.Description("Retain the release history")),
		mcp.WithBoolean("wait", mcp.Description("Wait for resource deletion to complete")),
		mcp.WithBoolean("disable_hooks", mcp.Description("Disable pre/post delete hooks")),
		mcp.WithNumber("timeout_seconds", mcp.Description("Operation timeout in seconds")),
		mcp.WithString("description", mcp.Description("Human-readable description for this operation")),
	)
}

// --- Extended release inspection tool definitions ---

func toolGetReleaseMetadata() mcp.Tool {
	return mcp.NewTool("get_release_metadata",
		mcp.WithDescription("Return structured metadata for a Helm release: chart info, labels, annotations, dependencies, and apply method."),
		mcp.WithString("release_name", mcp.Required(), mcp.Description("Release name")),
		mcp.WithString("namespace", mcp.Required(), mcp.Description("Release namespace")),
		mcp.WithNumber("version", mcp.Description("Revision number (0 = current)")),
	)
}

func toolGetReleaseStatusResources() mcp.Tool {
	return mcp.NewTool("get_release_status_resources",
		mcp.WithDescription("Return the status of a Helm release together with the live Kubernetes resources it manages."),
		mcp.WithString("release_name", mcp.Required(), mcp.Description("Release name")),
		mcp.WithString("namespace", mcp.Required(), mcp.Description("Release namespace")),
		mcp.WithNumber("version", mcp.Description("Revision number (0 = current)")),
	)
}

func toolGetReleaseHooks() mcp.Tool {
	return mcp.NewTool("get_release_hooks",
		mcp.WithDescription("List lifecycle hooks for a deployed Helm release, including their last execution status."),
		mcp.WithString("release_name", mcp.Required(), mcp.Description("Release name")),
		mcp.WithString("namespace", mcp.Required(), mcp.Description("Release namespace")),
	)
}

func toolGetReleaseRevision() mcp.Tool {
	return mcp.NewTool("get_release_revision",
		mcp.WithDescription("Retrieve a specific historical revision of a Helm release, including the manifest and values."),
		mcp.WithString("release_name", mcp.Required(), mcp.Description("Release name")),
		mcp.WithString("namespace", mcp.Required(), mcp.Description("Release namespace")),
		mcp.WithNumber("version", mcp.Required(), mcp.Description("Revision number to retrieve")),
	)
}

func toolListReleasesFiltered() mcp.Tool {
	return mcp.NewTool("list_releases_filtered",
		mcp.WithDescription("List Helm releases with advanced filtering by state, label selector, regex filter, and pagination/sort options."),
		mcp.WithString("namespace", mcp.Description("Filter by namespace (omit for all namespaces)")),
		mcp.WithBoolean("all_namespaces", mcp.Description("List releases across all namespaces")),
		mcp.WithString("filter", mcp.Description("Regex filter applied to release names")),
		mcp.WithString("selector", mcp.Description("Label selector (e.g. app=nginx)")),
		mcp.WithString("state_mask", mcp.Description("State filter: deployed, failed, uninstalled, uninstalling, pending, superseded, all")),
		mcp.WithNumber("limit", mcp.Description("Maximum number of results to return")),
		mcp.WithNumber("offset", mcp.Description("Starting index for pagination")),
		mcp.WithString("sort_by", mcp.Description("Sort field: date or name (default)")),
		mcp.WithBoolean("sort_reverse", mcp.Description("Reverse the sort order")),
	)
}

// --- Chart inspection tool definitions ---

func toolShowChartValues() mcp.Tool {
	return mcp.NewTool("show_chart_values",
		mcp.WithDescription("Show the default values.yaml for a Helm chart without installing it."),
		mcp.WithString("chart_name", mcp.Required(), mcp.Description("Chart name")),
		mcp.WithString("repo_url", mcp.Required(), mcp.Description("Helm repository URL")),
		mcp.WithString("version", mcp.Description("Chart version (omit for latest)")),
	)
}

func toolShowChartReadme() mcp.Tool {
	return mcp.NewTool("show_chart_readme",
		mcp.WithDescription("Show the README for a Helm chart without installing it."),
		mcp.WithString("chart_name", mcp.Required(), mcp.Description("Chart name")),
		mcp.WithString("repo_url", mcp.Required(), mcp.Description("Helm repository URL")),
		mcp.WithString("version", mcp.Description("Chart version (omit for latest)")),
	)
}

func toolShowChartCRDs() mcp.Tool {
	return mcp.NewTool("show_chart_crds",
		mcp.WithDescription("Show the CRD manifests bundled with a Helm chart without installing it."),
		mcp.WithString("chart_name", mcp.Required(), mcp.Description("Chart name")),
		mcp.WithString("repo_url", mcp.Required(), mcp.Description("Helm repository URL")),
		mcp.WithString("version", mcp.Description("Chart version (omit for latest)")),
	)
}

func toolListChartDependencies() mcp.Tool {
	return mcp.NewTool("list_chart_dependencies",
		mcp.WithDescription("List the chart dependencies declared in Chart.yaml: versions, repositories, conditions and tags."),
		mcp.WithString("chart_name", mcp.Required(), mcp.Description("Chart name")),
		mcp.WithString("repo_url", mcp.Required(), mcp.Description("Helm repository URL")),
		mcp.WithString("version", mcp.Description("Chart version (omit for latest)")),
	)
}

func toolTemplateChart() mcp.Tool {
	return mcp.NewTool("template_chart",
		mcp.WithDescription("Render a Helm chart's Kubernetes manifests locally without connecting to a cluster, equivalent to 'helm template'."),
		mcp.WithString("chart_name", mcp.Required(), mcp.Description("Chart name")),
		mcp.WithString("repo_url", mcp.Description("Helm repository URL")),
		mcp.WithString("namespace", mcp.Description("Namespace to use in rendered templates (default: default)")),
		mcp.WithString("release_name", mcp.Description("Release name to use in rendered templates (default: release-name)")),
		mcp.WithString("version", mcp.Description("Chart version (omit for latest)")),
		mcp.WithObject("values", mcp.Description("Values overrides")),
		mcp.WithBoolean("show_notes", mcp.Description("Include NOTES.txt in output")),
		mcp.WithBoolean("include_crds", mcp.Description("Include CRD manifests in output")),
		mcp.WithBoolean("skip_schema_validation", mcp.Description("Skip values JSON schema validation")),
	)
}

func toolLintChart() mcp.Tool {
	return mcp.NewTool("lint_chart",
		mcp.WithDescription("Lint local Helm chart directories, checking templates, values schema, metadata, and best practices."),
		mcp.WithString("chart_path", mcp.Required(), mcp.Description("Path to the local chart directory (or comma-separated list of paths)")),
		mcp.WithObject("values", mcp.Description("Values overrides to use during linting")),
		mcp.WithBoolean("strict", mcp.Description("Fail on warnings as well as errors")),
		mcp.WithString("namespace", mcp.Description("Namespace to use in lint context")),
		mcp.WithBoolean("skip_schema_validation", mcp.Description("Skip JSON schema validation")),
	)
}

// --- Chart distribution tool definitions ---

func toolPackageChart() mcp.Tool {
	return mcp.NewTool("package_chart",
		mcp.WithDescription("Package a local Helm chart directory into a versioned .tgz archive ready for distribution."),
		mcp.WithString("chart_path", mcp.Required(), mcp.Description("Path to the local chart directory")),
		mcp.WithString("version", mcp.Description("Override the chart version")),
		mcp.WithString("app_version", mcp.Description("Override the chart appVersion")),
		mcp.WithString("destination", mcp.Description("Directory to write the .tgz file to (default: current directory)")),
		mcp.WithBoolean("sign", mcp.Description("Sign the chart package")),
		mcp.WithString("key", mcp.Description("Name of the GPG key to sign with")),
		mcp.WithString("keyring", mcp.Description("Path to the keyring file")),
	)
}

func toolPullChart() mcp.Tool {
	return mcp.NewTool("pull_chart",
		mcp.WithDescription("Download a Helm chart from a repository or OCI registry to a local directory, optionally extracting it."),
		mcp.WithString("chart_ref", mcp.Required(), mcp.Description("Chart reference (name or OCI URI)")),
		mcp.WithString("version", mcp.Description("Chart version to download")),
		mcp.WithString("repo_url", mcp.Description("Helm repository URL")),
		mcp.WithString("dest_dir", mcp.Description("Destination directory (default: current directory)")),
		mcp.WithBoolean("untar", mcp.Description("Extract the chart archive after downloading")),
		mcp.WithString("untar_dir", mcp.Description("Directory to extract into")),
		mcp.WithString("username", mcp.Description("Repository username")),
		mcp.WithString("password", mcp.Description("Repository password")),
		mcp.WithString("ca_file", mcp.Description("Path to CA certificate")),
		mcp.WithString("cert_file", mcp.Description("Path to TLS client certificate")),
		mcp.WithString("key_file", mcp.Description("Path to TLS client key")),
		mcp.WithBoolean("insecure_skip_tls_verify", mcp.Description("Disable TLS certificate verification")),
		mcp.WithBoolean("pass_credentials_all", mcp.Description("Pass credentials to all hosts")),
		mcp.WithBoolean("plain_http", mcp.Description("Use plain HTTP instead of HTTPS")),
	)
}

func toolPushChart() mcp.Tool {
	return mcp.NewTool("push_chart",
		mcp.WithDescription("Push a local Helm chart archive to an OCI registry."),
		mcp.WithString("chart_path", mcp.Required(), mcp.Description("Path to the .tgz chart archive")),
		mcp.WithString("remote", mcp.Required(), mcp.Description("OCI registry URL (e.g. oci://registry.example.com/charts)")),
		mcp.WithString("ca_file", mcp.Description("Path to CA certificate for the registry")),
		mcp.WithString("cert_file", mcp.Description("Path to TLS client certificate")),
		mcp.WithString("key_file", mcp.Description("Path to TLS client key")),
		mcp.WithBoolean("insecure_skip_tls_verify", mcp.Description("Disable TLS certificate verification")),
		mcp.WithBoolean("plain_http", mcp.Description("Use plain HTTP instead of HTTPS")),
	)
}

func toolTestRelease() mcp.Tool {
	return mcp.NewTool("test_release",
		mcp.WithDescription("Run the test hooks (helm test) for a deployed Helm release and return the pass/fail status of each test pod."),
		mcp.WithString("release_name", mcp.Required(), mcp.Description("Release name to test")),
		mcp.WithString("namespace", mcp.Required(), mcp.Description("Release namespace")),
		mcp.WithNumber("timeout_seconds", mcp.Description("Test timeout in seconds")),
		mcp.WithString("filter", mcp.Description("Comma-separated list of test hook names to include")),
	)
}

// --- Repository management tool definitions (additional) ---

func toolRepoUpdateSingle() mcp.Tool {
	return mcp.NewTool("repo_update_single",
		mcp.WithDescription("Refresh the index for a single named Helm chart repository."),
		mcp.WithString("name", mcp.Required(), mcp.Description("Repository alias name to update")),
	)
}
