package mcp

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	appdeployment "github.com/vinaycharlie01/sh-mcp-go/internal/application/deployment"
	appcluster "github.com/vinaycharlie01/sh-mcp-go/internal/application/cluster"
	appplanner "github.com/vinaycharlie01/sh-mcp-go/internal/application/planner"
	"github.com/vinaycharlie01/sh-mcp-go/internal/infrastructure/config"
	"github.com/vinaycharlie01/sh-mcp-go/internal/ports/outbound"
)

// Server wraps the MCP server and registers all sh-mcp-go tools.
type Server struct {
	mcp        *server.MCPServer
	handler    *Handler
	cfg        *config.MCPConfig
	logger     *slog.Logger
}

// NewServer creates and configures the MCP server with all tools registered.
func NewServer(
	cfg *config.MCPConfig,
	deploymentSvc *appdeployment.Service,
	clusterSvc *appcluster.Service,
	plannerSvc *appplanner.Service,
	helmPort outbound.HelmPort,
	logger *slog.Logger,
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
		logger:        logger,
	}

	registerTools(s, h)

	return &Server{
		mcp:     s,
		handler: h,
		cfg:     cfg,
		logger:  logger,
	}
}

// ServeStdio runs the MCP server over stdin/stdout (standard MCP transport).
func (s *Server) ServeStdio(_ context.Context) error {
	s.logger.Info("starting MCP server (stdio transport)")

	return server.ServeStdio(s.mcp)
}

// ServeSSE runs the MCP server as an SSE HTTP endpoint.
func (s *Server) ServeSSE(_ context.Context) error {
	addr := s.cfg.SSEAddr
	if addr == "" {
		addr = "0.0.0.0:8081"
	}
	s.logger.Info("starting MCP server (SSE transport)", slog.String("addr", addr))
	sseServer := server.NewSSEServer(s.mcp, server.WithBaseURL(fmt.Sprintf("http://%s", addr)))
	return sseServer.Start(addr)
}

// registerTools registers all 24 MCP tools onto the server.
func registerTools(s *server.MCPServer, h *Handler) {
	// ---- Chart lifecycle ----
	s.AddTool(toolInstallChart(), h.InstallChart)
	s.AddTool(toolUpgradeChart(), h.UpgradeChart)
	s.AddTool(toolRollbackChart(), h.RollbackChart)
	s.AddTool(toolUninstallChart(), h.UninstallChart)

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
