package outbound

import (
	"context"

	"helm.sh/helm/v3/pkg/release"

	"github.com/vinaycharlie01/sh-mcp-go/internal/domain/deployment"
)

// HelmInstallRequest carries the parameters for a chart install.
type HelmInstallRequest struct {
	ReleaseName string
	Namespace   string
	ChartName   string
	RepoURL     string
	Version     string
	Values      map[string]any
	DryRun      bool
	Wait        bool
	WaitForJobs bool
	Timeout     int // seconds
	Atomic      bool
	CreateNS    bool
}

// HelmUpgradeRequest carries parameters for a chart upgrade.
type HelmUpgradeRequest struct {
	ReleaseName string
	Namespace   string
	ChartName   string
	RepoURL     string
	Version     string
	Values      map[string]any
	DryRun      bool
	Wait        bool
	Timeout     int
	Atomic      bool
	ReuseValues bool
	ResetValues bool
	Force       bool
}

// HelmRollbackRequest carries parameters for a chart rollback.
type HelmRollbackRequest struct {
	ReleaseName string
	Namespace   string
	Version     int // 0 means previous
	DryRun      bool
	Wait        bool
	Timeout     int
	Force       bool
}

// HelmUninstallRequest carries parameters for an uninstall.
type HelmUninstallRequest struct {
	ReleaseName string
	Namespace   string
	DryRun      bool
	KeepHistory bool
	Timeout     int
}

// HelmDiffResult contains the diff between current and desired release state.
type HelmDiffResult struct {
	HasChanges bool
	Diff       string
}

// HelmPort is the outbound port for all Helm SDK interactions.
// Implementations must NOT call the helm CLI binary.
type HelmPort interface {
	// Install installs a Helm chart and returns the resulting release.
	Install(ctx context.Context, req HelmInstallRequest) (*release.Release, error)

	// Upgrade upgrades an existing Helm release.
	Upgrade(ctx context.Context, req HelmUpgradeRequest) (*release.Release, error)

	// Rollback rolls back a Helm release to a specific revision.
	Rollback(ctx context.Context, req HelmRollbackRequest) error

	// Uninstall removes a Helm release from the cluster.
	Uninstall(ctx context.Context, req HelmUninstallRequest) error

	// GetRelease retrieves the current state of a Helm release.
	GetRelease(ctx context.Context, releaseName, namespace string) (*release.Release, error)

	// ListReleases lists all Helm releases, optionally filtered by namespace.
	ListReleases(ctx context.Context, namespace string) ([]*release.Release, error)

	// GetHistory returns the revision history of a release.
	GetHistory(ctx context.Context, releaseName, namespace string, maxRevisions int) ([]*release.Release, error)

	// DryRunInstall performs a dry-run install and returns the rendered manifests.
	DryRunInstall(ctx context.Context, req HelmInstallRequest) (string, error)

	// DryRunUpgrade performs a dry-run upgrade and returns rendered manifests.
	DryRunUpgrade(ctx context.Context, req HelmUpgradeRequest) (string, error)

	// Diff returns the diff between current release and desired state.
	Diff(ctx context.Context, req HelmUpgradeRequest) (*HelmDiffResult, error)

	// ValidateChart validates a chart's structure and values schema.
	ValidateChart(ctx context.Context, chartName, repoURL, version string) error

	// GenerateValues generates a values.yaml skeleton for a chart.
	GenerateValues(ctx context.Context, chartName, repoURL, version string) (map[string]any, error)

	// ResolveVersion resolves the latest stable version of a chart.
	ResolveVersion(ctx context.Context, chartName, repoURL, constraint string) (string, error)

	// BuildDependencies fetches and builds chart dependencies.
	BuildDependencies(ctx context.Context, chartName, repoURL, version string) error
}

// ReleaseMapper converts a helm release to our domain type.
func ReleaseToChartRef(r *release.Release) deployment.ChartReference {
	ver := ""
	if r.Chart != nil && r.Chart.Metadata != nil {
		ver = r.Chart.Metadata.Version
	}
	name := ""
	if r.Chart != nil && r.Chart.Metadata != nil {
		name = r.Chart.Metadata.Name
	}
	return deployment.ChartReference{
		Name:    name,
		Version: ver,
		Source:  deployment.ChartSourceRepo,
	}
}
