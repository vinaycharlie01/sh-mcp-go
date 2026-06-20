package deployment

import "github.com/vinaycharlie01/sh-mcp-go/internal/domain/deployment"

// InstallChartCommand carries intent to install a chart.
type InstallChartCommand struct {
	ReleaseName string
	Namespace   string
	ChartName   string
	RepoURL     string
	Version     string
	Values      map[string]any
	DryRun      bool
	Wait        bool
	Atomic      bool
	CreateNS    bool
	TimeoutSecs int
}

// UpgradeChartCommand carries intent to upgrade a chart.
type UpgradeChartCommand struct {
	ReleaseName string
	Namespace   string
	ChartName   string
	RepoURL     string
	Version     string
	Values      map[string]any
	DryRun      bool
	Wait        bool
	Atomic      bool
	ReuseValues bool
	ResetValues bool
	Force       bool
	TimeoutSecs int
}

// RollbackChartCommand carries intent to roll back a release.
type RollbackChartCommand struct {
	ReleaseName string
	Namespace   string
	Version     int
	DryRun      bool
	Wait        bool
	TimeoutSecs int
}

// UninstallChartCommand carries intent to uninstall a release.
type UninstallChartCommand struct {
	ReleaseName string
	Namespace   string
	DryRun      bool
	KeepHistory bool
	TimeoutSecs int
}

// InstallChartResult is returned from a successful install.
type InstallChartResult struct {
	DeploymentID string
	ReleaseName  string
	Namespace    string
	Revision     int
	Status       deployment.Status
	Notes        string
}

// UpgradeChartResult is returned from a successful upgrade.
type UpgradeChartResult struct {
	DeploymentID string
	ReleaseName  string
	Namespace    string
	Revision     int
	Status       deployment.Status
	Notes        string
}
