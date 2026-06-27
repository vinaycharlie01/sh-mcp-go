package outbound

import (
	"context"

	releasev1 "helm.sh/helm/v4/pkg/release/v1"

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
	// Advanced fields
	Labels                   map[string]string
	Description              string
	GenerateName             bool
	NameTemplate             string
	DisableHooks             bool
	Replace                  bool
	SkipCRDs                 bool
	SubNotes                 bool
	SkipSchemaValidation     bool
	DisableOpenAPIValidation bool
	ServerSideApply          bool
	ForceConflicts           bool
	TakeOwnership            bool
	IncludeCRDs              bool
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
	// Advanced fields
	Labels                   map[string]string
	Description              string
	DisableHooks             bool
	CleanupOnFail            bool
	MaxHistory               int
	RollbackOnFailure        bool
	ResetThenReuseValues     bool
	SkipSchemaValidation     bool
	DisableOpenAPIValidation bool
	ServerSideApply          string // "auto", "true", or "false"
	ForceConflicts           bool
	TakeOwnership            bool
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
	// Advanced fields
	DisableHooks    bool
	CleanupOnFail   bool
	MaxHistory      int
	ServerSideApply string // "auto", "true", or "false"
	ForceConflicts  bool
}

// HelmUninstallRequest carries parameters for an uninstall.
type HelmUninstallRequest struct {
	ReleaseName string
	Namespace   string
	DryRun      bool
	KeepHistory bool
	Timeout     int
	// Advanced fields
	Wait         bool
	DisableHooks bool
	Description  string
}

// HelmDiffResult contains the diff between current and desired release state.
type HelmDiffResult struct {
	HasChanges bool
	Diff       string
}

// RepoEntry describes a Helm chart repository configuration entry.
type RepoEntry struct {
	Name                  string
	URL                   string
	Username              string
	Password              string
	CAFile                string
	CertFile              string
	KeyFile               string
	InsecureSkipTLSVerify bool
	PassCredentialsAll    bool
}

// RepoSearchResult is a single chart hit from a repository search.
type RepoSearchResult struct {
	Name         string
	ChartVersion string
	AppVersion   string
	Description  string
	RepoURL      string
}

// ChartDetails contains chart metadata, default values and README content.
type ChartDetails struct {
	Metadata      map[string]any
	DefaultValues map[string]any
	Readme        string
}

// RegistryLoginRequest carries credentials for an OCI registry login.
type RegistryLoginRequest struct {
	Host                  string
	Username              string
	Password              string
	CertFile              string
	KeyFile               string
	CAFile                string
	InsecureSkipTLSVerify bool
	PlainHTTP             bool
}

// LintMessage is a single message from a chart lint run.
type LintMessage struct {
	Severity string
	Path     string
	Message  string
}

// LintResult contains the results of linting one or more chart directories.
type LintResult struct {
	TotalCharts int
	Messages    []*LintMessage
	Errors      []string
}

// PackageRequest carries the parameters for packaging a chart directory.
type PackageRequest struct {
	ChartPath   string
	Version     string
	AppVersion  string
	Destination string
	Sign        bool
	Key         string
	Keyring     string
}

// PullRequest carries the parameters for pulling a chart from a repository.
type PullRequest struct {
	ChartRef              string
	Version               string
	RepoURL               string
	DestDir               string
	Untar                 bool
	UntarDir              string
	Username              string
	Password              string
	CertFile              string
	KeyFile               string
	CAFile                string
	InsecureSkipTLSVerify bool
	PassCredentialsAll    bool
	PlainHTTP             bool
}

// PushRequest carries the parameters for pushing a chart to an OCI registry.
type PushRequest struct {
	ChartPath             string
	Remote                string
	CertFile              string
	KeyFile               string
	CAFile                string
	InsecureSkipTLSVerify bool
	PlainHTTP             bool
}

// TestResult contains the outcome of running helm test on a release.
type TestResult struct {
	ReleaseName string
	Namespace   string
	Status      string
	Passed      int
	Failed      int
	Messages    []string
}

// ReleaseMetadata contains structured metadata for a deployed release.
type ReleaseMetadata struct {
	Name         string
	Chart        string
	Version      string
	AppVersion   string
	Annotations  map[string]string
	Labels       map[string]string
	Dependencies []string
	Namespace    string
	Revision     int
	Status       string
	DeployedAt   string
	ApplyMethod  string
}

// ReleaseStatusDetails contains status and live resource information for a release.
type ReleaseStatusDetails struct {
	ReleaseName string
	Namespace   string
	Revision    int
	Status      string
	Notes       string
	DeployedAt  string
	Resources   map[string]any
}

// HookInfo describes a single lifecycle hook attached to a release.
type HookInfo struct {
	Name   string
	Kind   string
	Path   string
	Events []string
	Status string
	Weight int
}

// DependencyEntry describes a chart dependency declared in Chart.yaml.
type DependencyEntry struct {
	Name       string
	Version    string
	Repository string
	Condition  string
	Tags       []string
	Alias      string
}

// TemplateRequest carries the parameters for rendering chart templates locally.
type TemplateRequest struct {
	ReleaseName          string
	Namespace            string
	ChartName            string
	RepoURL              string
	Version              string
	Values               map[string]any
	ShowNotes            bool
	IncludeCRDs          bool
	SkipSchemaValidation bool
}

// HelmListRequest carries the parameters for listing releases with filtering and sorting.
type HelmListRequest struct {
	Namespace     string
	AllNamespaces bool
	Filter        string
	Selector      string
	StateMask     string // "deployed", "failed", "uninstalled", "all", etc.
	Limit         int
	Offset        int
	SortBy        string // "date" or "" (name)
	SortReverse   bool
}

// HelmPort is the outbound port for all Helm SDK interactions.
// Implementations must NOT call the helm CLI binary.
type HelmPort interface {
	// Install installs a Helm chart and returns the resulting release.
	Install(ctx context.Context, req HelmInstallRequest) (*releasev1.Release, error)

	// Upgrade upgrades an existing Helm release.
	Upgrade(ctx context.Context, req HelmUpgradeRequest) (*releasev1.Release, error)

	// Rollback rolls back a Helm release to a specific revision.
	Rollback(ctx context.Context, req HelmRollbackRequest) error

	// Uninstall removes a Helm release from the cluster.
	Uninstall(ctx context.Context, req HelmUninstallRequest) error

	// GetRelease retrieves the current state of a Helm release.
	GetRelease(ctx context.Context, releaseName, namespace string) (*releasev1.Release, error)

	// ListReleases lists all Helm releases, optionally filtered by namespace.
	ListReleases(ctx context.Context, namespace string) ([]*releasev1.Release, error)

	// GetHistory returns the revision history of a release.
	GetHistory(ctx context.Context, releaseName, namespace string, maxRevisions int) ([]*releasev1.Release, error)

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

	// GetReleaseValues returns the computed values for a deployed release.
	// When allValues is true, default values are merged in; otherwise only user-supplied values are returned.
	GetReleaseValues(ctx context.Context, releaseName, namespace string, allValues bool) (map[string]any, error)

	// GetReleaseNotes returns the notes produced by a release's chart.
	GetReleaseNotes(ctx context.Context, releaseName, namespace string) (string, error)

	// GetReleaseManifest returns the Kubernetes manifest generated by a release.
	GetReleaseManifest(ctx context.Context, releaseName, namespace string) (string, error)

	// ShowChart returns metadata, default values and README for a chart without installing it.
	ShowChart(ctx context.Context, chartName, repoURL, version string) (*ChartDetails, error)

	// AddRepo adds a Helm chart repository and downloads its index.
	AddRepo(ctx context.Context, entry RepoEntry) error

	// RemoveRepo removes a named Helm chart repository from local configuration.
	RemoveRepo(ctx context.Context, name string) error

	// UpdateRepos refreshes the index for all configured repositories.
	UpdateRepos(ctx context.Context) error

	// ListRepos returns all configured Helm chart repositories.
	ListRepos(ctx context.Context) ([]*RepoEntry, error)

	// SearchRepo searches configured repositories for charts matching keyword.
	SearchRepo(ctx context.Context, keyword, repoURL string) ([]*RepoSearchResult, error)

	// RegistryLogin authenticates with an OCI registry.
	RegistryLogin(ctx context.Context, req RegistryLoginRequest) error

	// RegistryLogout removes stored credentials for an OCI registry.
	RegistryLogout(ctx context.Context, host string) error

	// LintChart lints local chart directories and returns lint messages and errors.
	LintChart(ctx context.Context, paths []string, values map[string]any) (*LintResult, error)

	// PackageChart packages a chart directory into a versioned .tgz archive.
	PackageChart(ctx context.Context, req PackageRequest) (string, error)

	// PullChart downloads a chart from a repository or OCI registry to a local directory.
	PullChart(ctx context.Context, req PullRequest) (string, error)

	// PushChart pushes a local chart archive to an OCI registry.
	PushChart(ctx context.Context, req PushRequest) (string, error)

	// TestRelease runs the test hooks for a deployed release and returns the results.
	TestRelease(ctx context.Context, releaseName, namespace string, timeout int, filters []string) (*TestResult, error)

	// GetReleaseMetadata returns structured release metadata including labels, annotations and dependencies.
	GetReleaseMetadata(ctx context.Context, releaseName, namespace string, version int) (*ReleaseMetadata, error)

	// GetReleaseStatusWithResources returns release status with live Kubernetes resource details.
	GetReleaseStatusWithResources(ctx context.Context, releaseName, namespace string, version int) (*ReleaseStatusDetails, error)

	// GetReleaseHooks returns the lifecycle hook definitions for a deployed release.
	GetReleaseHooks(ctx context.Context, releaseName, namespace string) ([]*HookInfo, error)

	// ShowChartValues returns the default values declared in a chart's values.yaml.
	ShowChartValues(ctx context.Context, chartName, repoURL, version string) (map[string]any, error)

	// ShowChartReadme returns the README content for a chart.
	ShowChartReadme(ctx context.Context, chartName, repoURL, version string) (string, error)

	// ShowChartCRDs returns the CRD manifests bundled with a chart.
	ShowChartCRDs(ctx context.Context, chartName, repoURL, version string) ([]string, error)

	// TemplateChart renders chart templates locally without connecting to Kubernetes.
	TemplateChart(ctx context.Context, req TemplateRequest) (string, error)

	// ListChartDependencies returns the dependency list declared in a chart's Chart.yaml.
	ListChartDependencies(ctx context.Context, chartName, repoURL, version string) ([]*DependencyEntry, error)

	// UpdateRepo refreshes the index for a single named repository.
	UpdateRepo(ctx context.Context, name string) error

	// ListReleasesFiltered lists releases with advanced filter, sort and pagination options.
	ListReleasesFiltered(ctx context.Context, req HelmListRequest) ([]*releasev1.Release, error)

	// GetReleaseRevision retrieves a specific historical revision of a release.
	GetReleaseRevision(ctx context.Context, releaseName, namespace string, version int) (*releasev1.Release, error)
}

// ReleaseToChartRef converts a helm release to our domain type.
func ReleaseToChartRef(r *releasev1.Release) deployment.ChartReference {
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
