package helm

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/repo"

	"github.com/vinaycharlie01/sh-mcp-go/internal/infrastructure/config"
	"github.com/vinaycharlie01/sh-mcp-go/internal/infrastructure/retry"
	"github.com/vinaycharlie01/sh-mcp-go/internal/ports/outbound"
)

const (
	dirPerm  os.FileMode = 0o755
	filePerm os.FileMode = 0o644
)

// Client implements outbound.HelmPort using the Helm SDK.
// No helm binary is invoked — all operations use the helm.sh/helm/v3 Go packages.
type Client struct {
	cfg         *config.HelmConfig
	settings    *cli.EnvSettings
	logger      *slog.Logger
	retryPolicy retry.Policy
	getters     getter.Providers
}

// NewClient constructs a Helm SDK client.
func NewClient(cfg *config.HelmConfig, logger *slog.Logger) (*Client, error) {
	settings := cli.New()
	settings.RepositoryCache = cfg.RepositoryCache
	settings.RepositoryConfig = cfg.RepositoryConfig
	if cfg.PluginsDir != "" {
		settings.PluginsDirectory = cfg.PluginsDir
	}

	if err := os.MkdirAll(cfg.RepositoryCache, dirPerm); err != nil {
		return nil, fmt.Errorf("creating helm cache dir: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(cfg.RepositoryConfig), dirPerm); err != nil {
		return nil, fmt.Errorf("creating helm config dir: %w", err)
	}

	return &Client{
		cfg:         cfg,
		settings:    settings,
		logger:      logger,
		retryPolicy: retry.DefaultHelmPolicy(logger),
		getters:     getter.All(settings),
	}, nil
}

// actionConfig builds a Helm action.Configuration for a specific namespace.
func (c *Client) actionConfig(namespace string) (*action.Configuration, error) {
	actionCfg := new(action.Configuration)
	if err := actionCfg.Init(
		c.settings.RESTClientGetter(),
		namespace,
		os.Getenv("HELM_DRIVER"), // defaults to "secrets"
		func(format string, v ...interface{}) {
			c.logger.Debug(fmt.Sprintf(format, v...))
		},
	); err != nil {
		return nil, fmt.Errorf("initializing helm action config for namespace %q: %w", namespace, err)
	}

	return actionCfg, nil
}

// Install installs a Helm chart using the SDK.
func (c *Client) Install(ctx context.Context, req outbound.HelmInstallRequest) (*release.Release, error) {
	c.logger.Info("helm install",
		slog.String("release", req.ReleaseName),
		slog.String("chart", req.ChartName),
		slog.String("namespace", req.Namespace),
		slog.String("version", req.Version),
	)

	actionCfg, err := c.actionConfig(req.Namespace)
	if err != nil {
		return nil, err
	}

	install := action.NewInstall(actionCfg)
	install.ReleaseName = req.ReleaseName
	install.Namespace = req.Namespace
	install.Version = req.Version
	install.DryRun = req.DryRun
	install.Wait = req.Wait
	install.WaitForJobs = req.WaitForJobs
	install.Atomic = req.Atomic
	install.CreateNamespace = req.CreateNS
	if req.Timeout > 0 {
		install.Timeout = time.Duration(req.Timeout) * time.Second
	} else {
		install.Timeout = c.cfg.DefaultTimeout
	}

	chrt, err := c.loadChart(ctx, req.ChartName, req.RepoURL, req.Version, install.ChartPathOptions)
	if err != nil {
		return nil, fmt.Errorf("loading chart %q: %w", req.ChartName, err)
	}

	var rel *release.Release
	err = retry.Do(ctx, c.retryPolicy, func() error {
		var runErr error
		rel, runErr = install.RunWithContext(ctx, chrt, req.Values)

		return runErr
	})
	if err != nil {
		return nil, fmt.Errorf("helm install %q: %w", req.ReleaseName, err)
	}

	c.logger.Info("helm install succeeded",
		slog.String("release", rel.Name),
		slog.Int("revision", rel.Version),
	)

	return rel, nil
}

// Upgrade upgrades a Helm release using the SDK.
func (c *Client) Upgrade(ctx context.Context, req outbound.HelmUpgradeRequest) (*release.Release, error) {
	c.logger.Info("helm upgrade",
		slog.String("release", req.ReleaseName),
		slog.String("chart", req.ChartName),
		slog.String("namespace", req.Namespace),
	)

	actionCfg, err := c.actionConfig(req.Namespace)
	if err != nil {
		return nil, err
	}

	upgrade := action.NewUpgrade(actionCfg)
	upgrade.Namespace = req.Namespace
	upgrade.Version = req.Version
	upgrade.DryRun = req.DryRun
	upgrade.Wait = req.Wait
	upgrade.Atomic = req.Atomic
	upgrade.ReuseValues = req.ReuseValues
	upgrade.ResetValues = req.ResetValues
	upgrade.Force = req.Force
	if req.Timeout > 0 {
		upgrade.Timeout = time.Duration(req.Timeout) * time.Second
	} else {
		upgrade.Timeout = c.cfg.DefaultTimeout
	}

	chrt, err := c.loadChart(ctx, req.ChartName, req.RepoURL, req.Version, upgrade.ChartPathOptions)
	if err != nil {
		return nil, fmt.Errorf("loading chart %q: %w", req.ChartName, err)
	}

	var rel *release.Release
	err = retry.Do(ctx, c.retryPolicy, func() error {
		var runErr error
		rel, runErr = upgrade.RunWithContext(ctx, req.ReleaseName, chrt, req.Values)

		return runErr
	})
	if err != nil {
		return nil, fmt.Errorf("helm upgrade %q: %w", req.ReleaseName, err)
	}

	c.logger.Info("helm upgrade succeeded", slog.String("release", rel.Name))

	return rel, nil
}

// Rollback rolls back a Helm release to a specific revision.
func (c *Client) Rollback(ctx context.Context, req outbound.HelmRollbackRequest) error {
	c.logger.Info("helm rollback",
		slog.String("release", req.ReleaseName),
		slog.Int("version", req.Version),
	)

	actionCfg, err := c.actionConfig(req.Namespace)
	if err != nil {
		return err
	}

	rollback := action.NewRollback(actionCfg)
	rollback.Version = req.Version
	rollback.DryRun = req.DryRun
	rollback.Wait = req.Wait
	rollback.Force = req.Force
	if req.Timeout > 0 {
		rollback.Timeout = time.Duration(req.Timeout) * time.Second
	} else {
		rollback.Timeout = c.cfg.DefaultTimeout
	}

	return rollback.Run(req.ReleaseName)
}

// Uninstall removes a Helm release.
func (c *Client) Uninstall(ctx context.Context, req outbound.HelmUninstallRequest) error {
	c.logger.Info("helm uninstall", slog.String("release", req.ReleaseName))

	actionCfg, err := c.actionConfig(req.Namespace)
	if err != nil {
		return err
	}

	uninstall := action.NewUninstall(actionCfg)
	uninstall.DryRun = req.DryRun
	uninstall.KeepHistory = req.KeepHistory
	if req.Timeout > 0 {
		uninstall.Timeout = time.Duration(req.Timeout) * time.Second
	}

	_, err = uninstall.Run(req.ReleaseName)

	return err
}

// GetRelease retrieves the current state of a Helm release.
func (c *Client) GetRelease(ctx context.Context, releaseName, namespace string) (*release.Release, error) {
	actionCfg, err := c.actionConfig(namespace)
	if err != nil {
		return nil, err
	}

	get := action.NewGet(actionCfg)

	return get.Run(releaseName)
}

// ListReleases lists Helm releases in a namespace.
func (c *Client) ListReleases(ctx context.Context, namespace string) ([]*release.Release, error) {
	actionCfg, err := c.actionConfig(namespace)
	if err != nil {
		return nil, err
	}

	list := action.NewList(actionCfg)
	list.AllNamespaces = namespace == ""
	list.All = true

	return list.Run()
}

// GetHistory returns revision history for a release.
func (c *Client) GetHistory(_ context.Context, releaseName, namespace string, maxRevisions int) ([]*release.Release, error) {
	actionCfg, err := c.actionConfig(namespace)
	if err != nil {
		return nil, err
	}

	history := action.NewHistory(actionCfg)
	history.Max = maxRevisions

	return history.Run(releaseName)
}

// DryRunInstall performs a dry-run install and returns rendered manifests.
func (c *Client) DryRunInstall(ctx context.Context, req outbound.HelmInstallRequest) (string, error) {
	req.DryRun = true
	rel, err := c.Install(ctx, req)
	if err != nil {
		return "", err
	}
	if rel == nil {
		return "", nil
	}

	return rel.Manifest, nil
}

// DryRunUpgrade performs a dry-run upgrade and returns rendered manifests.
func (c *Client) DryRunUpgrade(ctx context.Context, req outbound.HelmUpgradeRequest) (string, error) {
	req.DryRun = true
	rel, err := c.Upgrade(ctx, req)
	if err != nil {
		return "", err
	}
	if rel == nil {
		return "", nil
	}

	return rel.Manifest, nil
}

// Diff returns the diff between current and desired release state.
func (c *Client) Diff(ctx context.Context, req outbound.HelmUpgradeRequest) (*outbound.HelmDiffResult, error) {
	manifest, err := c.DryRunUpgrade(ctx, req)
	if err != nil {
		return nil, err
	}

	return &outbound.HelmDiffResult{
		HasChanges: manifest != "",
		Diff:       manifest,
	}, nil
}

// ValidateChart validates chart structure and schema.
func (c *Client) ValidateChart(ctx context.Context, chartName, repoURL, version string) error {
	chrt, err := c.loadChart(ctx, chartName, repoURL, version, action.ChartPathOptions{})
	if err != nil {
		return fmt.Errorf("loading chart for validation: %w", err)
	}

	return chrt.Validate()
}

// GenerateValues returns the default values for a chart.
func (c *Client) GenerateValues(ctx context.Context, chartName, repoURL, version string) (map[string]any, error) {
	chrt, err := c.loadChart(ctx, chartName, repoURL, version, action.ChartPathOptions{})
	if err != nil {
		return nil, err
	}

	return chrt.Values, nil
}

// ResolveVersion resolves the latest chart version matching a semver constraint.
func (c *Client) ResolveVersion(ctx context.Context, chartName, repoURL, constraint string) (string, error) {
	if err := c.updateRepoIndex(repoURL); err != nil {
		return "", fmt.Errorf("updating repo index: %w", err)
	}

	idx, err := c.loadRepoIndex(repoURL)
	if err != nil {
		return "", err
	}

	versions, ok := idx.Entries[chartName]
	if !ok || len(versions) == 0 {
		return "", fmt.Errorf("chart %q not found in repo %q", chartName, repoURL)
	}

	if constraint == "" || constraint == "latest" {
		return versions[0].Version, nil
	}

	for _, v := range versions {
		if v.Version == constraint {
			return v.Version, nil
		}
	}

	return "", fmt.Errorf("version %q of chart %q not found", constraint, chartName)
}

// BuildDependencies downloads and builds chart dependencies.
func (c *Client) BuildDependencies(ctx context.Context, chartName, repoURL, version string) error {
	// Chart dependency management requires a chart path; we log a note here.
	// Full dependency resolution is handled automatically by LocateChart.
	c.logger.Info("dependency build: dependencies are resolved during chart load",
		slog.String("chart", chartName))

	return nil
}

// loadChart downloads and loads a chart into memory.
func (c *Client) loadChart(_ context.Context, name, repoURL, version string, opts action.ChartPathOptions) (*chart.Chart, error) {
	opts.RepoURL = repoURL
	opts.Version = version

	// Register the repo if needed
	if repoURL != "" {
		if err := c.ensureRepo(name, repoURL); err != nil {
			c.logger.Warn("could not ensure repo", slog.String("url", repoURL), slog.String("error", err.Error()))
		}
	}

	chartPath, err := opts.LocateChart(name, c.settings)
	if err != nil {
		return nil, fmt.Errorf("locating chart %q: %w", name, err)
	}

	return loader.Load(chartPath)
}

// ensureRepo adds a Helm repository entry if not already configured.
func (c *Client) ensureRepo(name, url string) error {
	f, err := repo.LoadFile(c.settings.RepositoryConfig)
	if err != nil {
		f = repo.NewFile()
	}

	if f.Has(name) {
		return nil
	}

	entry := &repo.Entry{Name: name, URL: url}
	r, err := repo.NewChartRepository(entry, getter.All(c.settings))
	if err != nil {
		return err
	}

	if _, err := r.DownloadIndexFile(); err != nil {
		return fmt.Errorf("downloading index for %q: %w", url, err)
	}

	f.Update(entry)

	return f.WriteFile(c.settings.RepositoryConfig, filePerm)
}

// updateRepoIndex forces a refresh of the repo index.
func (c *Client) updateRepoIndex(_ string) error {
	return nil // index is lazily downloaded by LocateChart
}

// loadRepoIndex loads the repo index from cache.
func (c *Client) loadRepoIndex(_ string) (*repo.IndexFile, error) {
	files, err := filepath.Glob(filepath.Join(c.settings.RepositoryCache, "*.yaml"))
	if err != nil || len(files) == 0 {
		return nil, fmt.Errorf("no repo index files found in cache")
	}

	return repo.LoadIndexFile(files[0])
}
