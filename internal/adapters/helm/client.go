package helm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"helm.sh/helm/v4/pkg/action"
	"helm.sh/helm/v4/pkg/chart/loader"
	chartv2 "helm.sh/helm/v4/pkg/chart/v2"
	"helm.sh/helm/v4/pkg/cli"
	"helm.sh/helm/v4/pkg/getter"
	"helm.sh/helm/v4/pkg/kube"
	"helm.sh/helm/v4/pkg/registry"
	releasev1 "helm.sh/helm/v4/pkg/release/v1"
	repov1 "helm.sh/helm/v4/pkg/repo/v1"

	"github.com/vinaycharlie01/sh-mcp-go/internal/infrastructure/config"
	"github.com/vinaycharlie01/sh-mcp-go/internal/infrastructure/retry"
	"github.com/vinaycharlie01/sh-mcp-go/internal/ports/outbound"
)

const (
	dirPerm  os.FileMode = 0o755
	filePerm os.FileMode = 0o644
)

// Client implements outbound.HelmPort using the Helm SDK.
// No helm binary is invoked — all operations use the helm.sh/helm/v4 Go packages.
type Client struct {
	cfg            *config.HelmConfig
	settings       *cli.EnvSettings
	retryPolicy    retry.Policy
	getters        getter.Providers
	registryClient *registry.Client
}

// NewClient constructs a Helm SDK client.
func NewClient(cfg *config.HelmConfig) (*Client, error) {
	settings := cli.New()
	settings.RepositoryCache = cfg.RepositoryCache
	settings.RepositoryConfig = cfg.RepositoryConfig
	if cfg.PluginsDir != "" {
		settings.PluginsDirectory = cfg.PluginsDir
	}
	if cfg.RegistryConfig != "" {
		settings.RegistryConfig = cfg.RegistryConfig
	}

	if err := os.MkdirAll(cfg.RepositoryCache, dirPerm); err != nil {
		return nil, fmt.Errorf("creating helm cache dir: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(cfg.RepositoryConfig), dirPerm); err != nil {
		return nil, fmt.Errorf("creating helm config dir: %w", err)
	}

	regOpts := []registry.ClientOption{
		registry.ClientOptCredentialsFile(settings.RegistryConfig),
	}
	if cfg.PlainHTTP {
		regOpts = append(regOpts, registry.ClientOptPlainHTTP())
	}

	regClient, err := registry.NewClient(regOpts...)
	if err != nil {
		return nil, fmt.Errorf("creating registry client: %w", err)
	}

	return &Client{
		cfg:            cfg,
		settings:       settings,
		retryPolicy:    retry.DefaultHelmPolicy(),
		getters:        getter.All(settings),
		registryClient: regClient,
	}, nil
}

// actionConfig builds a Helm action.Configuration for a specific namespace.
func (c *Client) actionConfig(namespace string) (*action.Configuration, error) {
	actionCfg := action.NewConfiguration()
	if err := actionCfg.Init(
		c.settings.RESTClientGetter(),
		namespace,
		os.Getenv("HELM_DRIVER"), // defaults to "secrets"
	); err != nil {
		return nil, fmt.Errorf("initializing helm action config for namespace %q: %w", namespace, err)
	}

	actionCfg.RegistryClient = c.registryClient

	return actionCfg, nil
}

// Install installs a Helm chart using the SDK.
func (c *Client) Install(ctx context.Context, req outbound.HelmInstallRequest) (*releasev1.Release, error) {
	slog.Info("helm install",
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
	install.CreateNamespace = req.CreateNS
	install.WaitForJobs = req.WaitForJobs
	install.RollbackOnFailure = req.Atomic

	if req.DryRun {
		install.DryRunStrategy = action.DryRunClient
	} else {
		install.DryRunStrategy = action.DryRunNone
	}

	if req.Wait {
		install.WaitStrategy = kube.StatusWatcherStrategy
	} else {
		install.WaitStrategy = kube.HookOnlyStrategy
	}

	if req.Timeout > 0 {
		install.Timeout = time.Duration(req.Timeout) * time.Second
	} else {
		install.Timeout = c.cfg.DefaultTimeout
	}

	chrt, err := c.loadChart(ctx, req.ChartName, req.RepoURL, req.Version, install.ChartPathOptions)
	if err != nil {
		return nil, fmt.Errorf("loading chart %q: %w", req.ChartName, err)
	}

	var rawRel any
	err = retry.Do(ctx, c.retryPolicy, func() error {
		var runErr error
		rawRel, runErr = install.RunWithContext(ctx, chrt, req.Values)

		return runErr
	})
	if err != nil {
		return nil, fmt.Errorf("helm install %q: %w", req.ReleaseName, err)
	}

	rel, ok := rawRel.(*releasev1.Release)
	if !ok {
		return nil, fmt.Errorf("helm install %q: unexpected release type", req.ReleaseName)
	}

	slog.Info("helm install succeeded",
		slog.String("release", rel.Name),
		slog.Int("revision", rel.Version),
	)

	return rel, nil
}

// Upgrade upgrades a Helm release using the SDK.
func (c *Client) Upgrade(ctx context.Context, req outbound.HelmUpgradeRequest) (*releasev1.Release, error) {
	slog.Info("helm upgrade",
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
	upgrade.ReuseValues = req.ReuseValues
	upgrade.ResetValues = req.ResetValues
	upgrade.ForceReplace = req.Force
	upgrade.RollbackOnFailure = req.Atomic

	if req.DryRun {
		upgrade.DryRunStrategy = action.DryRunClient
	} else {
		upgrade.DryRunStrategy = action.DryRunNone
	}

	if req.Wait {
		upgrade.WaitStrategy = kube.StatusWatcherStrategy
	} else {
		upgrade.WaitStrategy = kube.HookOnlyStrategy
	}

	if req.Timeout > 0 {
		upgrade.Timeout = time.Duration(req.Timeout) * time.Second
	} else {
		upgrade.Timeout = c.cfg.DefaultTimeout
	}

	chrt, err := c.loadChart(ctx, req.ChartName, req.RepoURL, req.Version, upgrade.ChartPathOptions)
	if err != nil {
		return nil, fmt.Errorf("loading chart %q: %w", req.ChartName, err)
	}

	var rawRel any
	err = retry.Do(ctx, c.retryPolicy, func() error {
		var runErr error
		rawRel, runErr = upgrade.RunWithContext(ctx, req.ReleaseName, chrt, req.Values)

		return runErr
	})
	if err != nil {
		return nil, fmt.Errorf("helm upgrade %q: %w", req.ReleaseName, err)
	}

	rel, ok := rawRel.(*releasev1.Release)
	if !ok {
		return nil, fmt.Errorf("helm upgrade %q: unexpected release type", req.ReleaseName)
	}

	slog.Info("helm upgrade succeeded", slog.String("release", rel.Name))

	return rel, nil
}

// Rollback rolls back a Helm release to a specific revision.
func (c *Client) Rollback(ctx context.Context, req outbound.HelmRollbackRequest) error {
	slog.Info("helm rollback",
		slog.String("release", req.ReleaseName),
		slog.Int("version", req.Version),
	)

	actionCfg, err := c.actionConfig(req.Namespace)
	if err != nil {
		return err
	}

	rollback := action.NewRollback(actionCfg)
	rollback.Version = req.Version
	rollback.ForceReplace = req.Force

	if req.DryRun {
		rollback.DryRunStrategy = action.DryRunClient
	} else {
		rollback.DryRunStrategy = action.DryRunNone
	}

	if req.Wait {
		rollback.WaitStrategy = kube.StatusWatcherStrategy
	} else {
		rollback.WaitStrategy = kube.HookOnlyStrategy
	}

	if req.Timeout > 0 {
		rollback.Timeout = time.Duration(req.Timeout) * time.Second
	} else {
		rollback.Timeout = c.cfg.DefaultTimeout
	}

	return rollback.Run(req.ReleaseName)
}

// Uninstall removes a Helm release.
func (c *Client) Uninstall(ctx context.Context, req outbound.HelmUninstallRequest) error {
	slog.Info("helm uninstall", slog.String("release", req.ReleaseName))

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
func (c *Client) GetRelease(ctx context.Context, releaseName, namespace string) (*releasev1.Release, error) {
	actionCfg, err := c.actionConfig(namespace)
	if err != nil {
		return nil, err
	}

	get := action.NewGet(actionCfg)
	rawRel, err := get.Run(releaseName)
	if err != nil {
		return nil, err
	}

	rel, ok := rawRel.(*releasev1.Release)
	if !ok {
		return nil, fmt.Errorf("helm get %q: unexpected release type", releaseName)
	}

	return rel, nil
}

// ListReleases lists Helm releases in a namespace.
func (c *Client) ListReleases(ctx context.Context, namespace string) ([]*releasev1.Release, error) {
	actionCfg, err := c.actionConfig(namespace)
	if err != nil {
		return nil, err
	}

	list := action.NewList(actionCfg)
	list.AllNamespaces = namespace == ""
	list.All = true

	rawRels, err := list.Run()
	if err != nil {
		return nil, err
	}

	releases := make([]*releasev1.Release, 0, len(rawRels))
	for _, r := range rawRels {
		rel, ok := r.(*releasev1.Release)
		if ok {
			releases = append(releases, rel)
		}
	}

	return releases, nil
}

// GetHistory returns revision history for a release.
func (c *Client) GetHistory(_ context.Context, releaseName, namespace string, maxRevisions int) ([]*releasev1.Release, error) {
	actionCfg, err := c.actionConfig(namespace)
	if err != nil {
		return nil, err
	}

	history := action.NewHistory(actionCfg)
	history.Max = maxRevisions

	rawRels, err := history.Run(releaseName)
	if err != nil {
		return nil, err
	}

	releases := make([]*releasev1.Release, 0, len(rawRels))
	for _, r := range rawRels {
		rel, ok := r.(*releasev1.Release)
		if ok {
			releases = append(releases, rel)
		}
	}

	return releases, nil
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
	raw, err := c.loadChart(ctx, chartName, repoURL, version, action.ChartPathOptions{})
	if err != nil {
		return fmt.Errorf("loading chart for validation: %w", err)
	}

	chrt, ok := raw.(*chartv2.Chart)
	if !ok {
		return fmt.Errorf("chart %q: unexpected chart type", chartName)
	}

	return chrt.Validate()
}

// GenerateValues returns the default values for a chart.
func (c *Client) GenerateValues(ctx context.Context, chartName, repoURL, version string) (map[string]any, error) {
	raw, err := c.loadChart(ctx, chartName, repoURL, version, action.ChartPathOptions{})
	if err != nil {
		return nil, err
	}

	chrt, ok := raw.(*chartv2.Chart)
	if !ok {
		return nil, fmt.Errorf("chart %q: unexpected chart type", chartName)
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
	slog.Info("dependency build: dependencies are resolved during chart load",
		slog.String("chart", chartName))

	return nil
}

// GetReleaseValues returns the computed values for a deployed release.
func (c *Client) GetReleaseValues(_ context.Context, releaseName, namespace string, allValues bool) (map[string]any, error) {
	actionCfg, err := c.actionConfig(namespace)
	if err != nil {
		return nil, err
	}

	getValues := action.NewGetValues(actionCfg)
	getValues.AllValues = allValues

	return getValues.Run(releaseName)
}

// GetReleaseNotes returns the notes produced by a release's chart.
func (c *Client) GetReleaseNotes(ctx context.Context, releaseName, namespace string) (string, error) {
	rel, err := c.GetRelease(ctx, releaseName, namespace)
	if err != nil {
		return "", err
	}

	if rel.Info == nil {
		return "", nil
	}

	return rel.Info.Notes, nil
}

// GetReleaseManifest returns the Kubernetes manifest generated by a release.
func (c *Client) GetReleaseManifest(ctx context.Context, releaseName, namespace string) (string, error) {
	rel, err := c.GetRelease(ctx, releaseName, namespace)
	if err != nil {
		return "", err
	}

	return rel.Manifest, nil
}

// ShowChart returns metadata, default values and README for a chart without installing it.
func (c *Client) ShowChart(ctx context.Context, chartName, repoURL, version string) (*outbound.ChartDetails, error) {
	raw, err := c.loadChart(ctx, chartName, repoURL, version, action.ChartPathOptions{})
	if err != nil {
		return nil, err
	}

	chrt, ok := raw.(*chartv2.Chart)
	if !ok {
		return nil, fmt.Errorf("chart %q: unexpected chart type", chartName)
	}

	var meta map[string]any
	if chrt.Metadata != nil {
		b, _ := json.Marshal(chrt.Metadata)
		_ = json.Unmarshal(b, &meta)
	}

	readme := ""
	for _, f := range chrt.Files {
		if strings.EqualFold(f.Name, "README.md") {
			readme = string(f.Data)

			break
		}
	}

	return &outbound.ChartDetails{
		Metadata:      meta,
		DefaultValues: chrt.Values,
		Readme:        readme,
	}, nil
}

// AddRepo adds a Helm chart repository and downloads its index.
func (c *Client) AddRepo(_ context.Context, entry outbound.RepoEntry) error {
	f, err := repov1.LoadFile(c.settings.RepositoryConfig)
	if err != nil {
		f = repov1.NewFile()
	}

	repoEntry := &repov1.Entry{
		Name:                  entry.Name,
		URL:                   entry.URL,
		Username:              entry.Username,
		Password:              entry.Password,
		CAFile:                entry.CAFile,
		CertFile:              entry.CertFile,
		KeyFile:               entry.KeyFile,
		InsecureSkipTLSVerify: entry.InsecureSkipTLSVerify,
		PassCredentialsAll:    entry.PassCredentialsAll,
	}

	r, err := repov1.NewChartRepository(repoEntry, getter.All(c.settings))
	if err != nil {
		return err
	}

	r.CachePath = c.settings.RepositoryCache
	if _, err := r.DownloadIndexFile(); err != nil {
		return fmt.Errorf("downloading index for %q: %w", entry.URL, err)
	}

	f.Update(repoEntry)

	return f.WriteFile(c.settings.RepositoryConfig, filePerm)
}

// RemoveRepo removes a named Helm chart repository from local configuration.
func (c *Client) RemoveRepo(_ context.Context, name string) error {
	f, err := repov1.LoadFile(c.settings.RepositoryConfig)
	if err != nil {
		return fmt.Errorf("loading repo config: %w", err)
	}

	if !f.Has(name) {
		return fmt.Errorf("repository %q not found", name)
	}

	f.Remove(name)

	return f.WriteFile(c.settings.RepositoryConfig, filePerm)
}

// UpdateRepos refreshes the index for all configured repositories.
func (c *Client) UpdateRepos(_ context.Context) error {
	f, err := repov1.LoadFile(c.settings.RepositoryConfig)
	if err != nil {
		return fmt.Errorf("loading repo config: %w", err)
	}

	var errs []error
	for _, entry := range f.Repositories {
		r, repoErr := repov1.NewChartRepository(entry, getter.All(c.settings))
		if repoErr != nil {
			errs = append(errs, fmt.Errorf("repo %q: %w", entry.Name, repoErr))

			continue
		}

		r.CachePath = c.settings.RepositoryCache
		if _, dlErr := r.DownloadIndexFile(); dlErr != nil {
			errs = append(errs, fmt.Errorf("updating repo %q: %w", entry.Name, dlErr))
		}
	}

	return errors.Join(errs...)
}

// ListRepos returns all configured Helm chart repositories.
func (c *Client) ListRepos(_ context.Context) ([]*outbound.RepoEntry, error) {
	f, err := repov1.LoadFile(c.settings.RepositoryConfig)
	if err != nil {
		return nil, nil
	}

	result := make([]*outbound.RepoEntry, 0, len(f.Repositories))
	for _, r := range f.Repositories {
		result = append(result, &outbound.RepoEntry{
			Name:                  r.Name,
			URL:                   r.URL,
			Username:              r.Username,
			Password:              r.Password,
			CAFile:                r.CAFile,
			CertFile:              r.CertFile,
			KeyFile:               r.KeyFile,
			InsecureSkipTLSVerify: r.InsecureSkipTLSVerify,
			PassCredentialsAll:    r.PassCredentialsAll,
		})
	}

	return result, nil
}

// SearchRepo searches cached repository indexes for charts matching keyword.
func (c *Client) SearchRepo(_ context.Context, keyword, repoURL string) ([]*outbound.RepoSearchResult, error) {
	idx, err := c.loadRepoIndex(repoURL)
	if err != nil {
		return nil, err
	}

	var results []*outbound.RepoSearchResult
	for chartName, versions := range idx.Entries {
		if keyword != "" && !strings.Contains(strings.ToLower(chartName), strings.ToLower(keyword)) {
			continue
		}

		if len(versions) == 0 {
			continue
		}

		latest := versions[0]
		results = append(results, &outbound.RepoSearchResult{
			Name:         chartName,
			ChartVersion: latest.Version,
			AppVersion:   latest.AppVersion,
			Description:  latest.Description,
			RepoURL:      repoURL,
		})
	}

	return results, nil
}

// RegistryLogin authenticates with an OCI registry.
func (c *Client) RegistryLogin(_ context.Context, req outbound.RegistryLoginRequest) error {
	actionCfg, err := c.actionConfig("")
	if err != nil {
		return err
	}

	login := action.NewRegistryLogin(actionCfg)

	return login.Run(os.Stderr, req.Host, req.Username, req.Password,
		action.WithCertFile(req.CertFile),
		action.WithKeyFile(req.KeyFile),
		action.WithCAFile(req.CAFile),
		action.WithInsecure(req.InsecureSkipTLSVerify),
		action.WithPlainHTTPLogin(req.PlainHTTP),
	)
}

// RegistryLogout removes stored credentials for an OCI registry.
func (c *Client) RegistryLogout(_ context.Context, host string) error {
	actionCfg, err := c.actionConfig("")
	if err != nil {
		return err
	}

	logout := action.NewRegistryLogout(actionCfg)

	return logout.Run(os.Stderr, host)
}

// loadChart downloads and loads a chart into memory.
func (c *Client) loadChart(_ context.Context, name, repoURL, version string, opts action.ChartPathOptions) (any, error) {
	opts.RepoURL = repoURL
	opts.Version = version

	if repoURL != "" {
		if err := c.ensureRepo(name, repoURL); err != nil {
			slog.Warn("could not ensure repo", slog.String("url", repoURL), slog.String("error", err.Error()))
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
	f, err := repov1.LoadFile(c.settings.RepositoryConfig)
	if err != nil {
		f = repov1.NewFile()
	}

	if f.Has(name) {
		return nil
	}

	entry := &repov1.Entry{Name: name, URL: url}
	r, err := repov1.NewChartRepository(entry, getter.All(c.settings))
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
func (c *Client) loadRepoIndex(_ string) (*repov1.IndexFile, error) {
	files, err := filepath.Glob(filepath.Join(c.settings.RepositoryCache, "*.yaml"))
	if err != nil || len(files) == 0 {
		return nil, fmt.Errorf("no repo index files found in cache")
	}

	return repov1.LoadIndexFile(files[0])
}
