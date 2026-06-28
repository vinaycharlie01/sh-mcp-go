//go:build integration

package helm_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go/modules/k3s"

	helmadapter "github.com/vinaycharlie01/sh-mcp-go/internal/adapters/helm"
	"github.com/vinaycharlie01/sh-mcp-go/internal/infrastructure/config"
	"github.com/vinaycharlie01/sh-mcp-go/internal/ports/outbound"
)

// suite holds shared state for all integration tests in this file.
var suite struct {
	client     *helmadapter.Client
	chartPath  string // absolute path to testdata/charts/hello
	kubecfgTmp string // temp kubeconfig written from k3s container
}

// TestMain spins up a k3s testcontainer once for the whole integration suite,
// creates a Helm client pointing at it, runs all tests, then tears everything down.
func TestMain(m *testing.M) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// ── 1. Start k3s ──────────────────────────────────────────────────────────
	k3sContainer, err := k3s.Run(ctx, "rancher/k3s:v1.31.4-k3s1")
	if err != nil {
		panic("k3s.Run: " + err.Error())
	}
	defer func() { _ = k3sContainer.Terminate(context.Background()) }()

	// ── 2. Write kubeconfig to a temp file ────────────────────────────────────
	kubeconfig, err := k3sContainer.GetKubeConfig(ctx)
	if err != nil {
		panic("GetKubeConfig: " + err.Error())
	}

	kubecfgFile, err := os.CreateTemp("", "k3s-kubeconfig-*.yaml")
	if err != nil {
		panic("create kubeconfig temp file: " + err.Error())
	}
	if _, err = kubecfgFile.Write(kubeconfig); err != nil {
		panic("write kubeconfig: " + err.Error())
	}
	kubecfgFile.Close()
	suite.kubecfgTmp = kubecfgFile.Name()
	defer os.Remove(suite.kubecfgTmp)

	// The Helm CLI settings pick up KUBECONFIG from the environment.
	os.Setenv("KUBECONFIG", suite.kubecfgTmp)

	// ── 3. Create Helm client ─────────────────────────────────────────────────
	dir, err := os.MkdirTemp("", "helm-integration-*")
	if err != nil {
		panic("create helm temp dir: " + err.Error())
	}
	defer os.RemoveAll(dir)

	helmCfg := &config.HelmConfig{
		RepositoryCache:  filepath.Join(dir, "cache"),
		RepositoryConfig: filepath.Join(dir, "repositories.yaml"),
		DefaultTimeout:   2 * time.Minute,
	}
	suite.client, err = helmadapter.NewClient(helmCfg)
	if err != nil {
		panic("NewClient: " + err.Error())
	}

	// ── 4. Resolve testdata chart path ────────────────────────────────────────
	// __FILE__ is in internal/adapters/helm — testdata lives next to it.
	wd, err := os.Getwd()
	if err != nil {
		panic("os.Getwd: " + err.Error())
	}
	suite.chartPath = filepath.Join(wd, "testdata", "charts", "hello")

	// ── 5. Run tests ──────────────────────────────────────────────────────────
	os.Exit(m.Run())
}

// ─── helpers ──────────────────────────────────────────────────────────────────

// installHello installs the embedded hello chart and registers cleanup.
func installHello(t *testing.T, releaseName string) {
	t.Helper()
	ctx := context.Background()

	_, err := suite.client.Install(ctx, outbound.HelmInstallRequest{
		ReleaseName: releaseName,
		Namespace:   "default",
		ChartName:   suite.chartPath,
		Wait:        true,
		Timeout:     60,
	})
	if err != nil {
		t.Fatalf("Install(%q): %v", releaseName, err)
	}

	t.Cleanup(func() {
		_ = suite.client.Uninstall(context.Background(), outbound.HelmUninstallRequest{
			ReleaseName: releaseName,
			Namespace:   "default",
		})
	})
}

// ─── tests ────────────────────────────────────────────────────────────────────

// TestIntegration_Install verifies that a chart can be installed and then
// queried via GetRelease, and that the release status is "deployed".
func TestIntegration_Install(t *testing.T) {
	ctx := context.Background()
	const name = "hello-install"

	installHello(t, name)

	rel, err := suite.client.GetRelease(ctx, name, "default")
	if err != nil {
		t.Fatalf("GetRelease: %v", err)
	}
	if rel.Name != name {
		t.Errorf("release name = %q, want %q", rel.Name, name)
	}
	if rel.Info == nil || rel.Info.Status.String() != "deployed" {
		t.Errorf("release status = %v, want deployed", rel.Info.Status)
	}
}

// TestIntegration_ListReleases verifies that installed releases appear in the list.
func TestIntegration_ListReleases(t *testing.T) {
	ctx := context.Background()
	const name = "hello-list"

	installHello(t, name)

	releases, err := suite.client.ListReleases(ctx, "default")
	if err != nil {
		t.Fatalf("ListReleases: %v", err)
	}

	found := false
	for _, r := range releases {
		if r.Name == name {
			found = true

			break
		}
	}
	if !found {
		t.Errorf("release %q not found in ListReleases result (%d releases)", name, len(releases))
	}
}

// TestIntegration_Upgrade verifies that an installed release can be upgraded
// (bumping a value) and the new revision is reflected.
func TestIntegration_Upgrade(t *testing.T) {
	ctx := context.Background()
	const name = "hello-upgrade"

	installHello(t, name)

	rel, err := suite.client.Upgrade(ctx, outbound.HelmUpgradeRequest{
		ReleaseName: name,
		Namespace:   "default",
		ChartName:   suite.chartPath,
		Values:      map[string]any{"greeting": "hi"},
		Wait:        true,
		Timeout:     60,
	})
	if err != nil {
		t.Fatalf("Upgrade: %v", err)
	}
	if rel.Version != 2 {
		t.Errorf("after upgrade revision = %d, want 2", rel.Version)
	}
}

// TestIntegration_Rollback verifies that a release can be rolled back to
// revision 1 after an upgrade.
func TestIntegration_Rollback(t *testing.T) {
	ctx := context.Background()
	const name = "hello-rollback"

	installHello(t, name)

	// Upgrade to create revision 2.
	if _, err := suite.client.Upgrade(ctx, outbound.HelmUpgradeRequest{
		ReleaseName: name,
		Namespace:   "default",
		ChartName:   suite.chartPath,
		Wait:        true,
		Timeout:     60,
	}); err != nil {
		t.Fatalf("Upgrade (pre-rollback): %v", err)
	}

	// Roll back to revision 1.
	if err := suite.client.Rollback(ctx, outbound.HelmRollbackRequest{
		ReleaseName: name,
		Namespace:   "default",
		Version:     1,
		Wait:        true,
		Timeout:     60,
	}); err != nil {
		t.Fatalf("Rollback: %v", err)
	}

	rel, err := suite.client.GetRelease(ctx, name, "default")
	if err != nil {
		t.Fatalf("GetRelease after rollback: %v", err)
	}
	if rel.Info == nil || rel.Info.Status.String() != "deployed" {
		t.Errorf("status after rollback = %v, want deployed", rel.Info.Status)
	}
}

// TestIntegration_GetReleaseValues verifies that user-supplied values
// are retrievable after install.
func TestIntegration_GetReleaseValues(t *testing.T) {
	ctx := context.Background()
	const name = "hello-values"

	_, err := suite.client.Install(ctx, outbound.HelmInstallRequest{
		ReleaseName: name,
		Namespace:   "default",
		ChartName:   suite.chartPath,
		Values:      map[string]any{"custom": "value"},
		Wait:        true,
		Timeout:     60,
	})
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	t.Cleanup(func() {
		_ = suite.client.Uninstall(context.Background(), outbound.HelmUninstallRequest{
			ReleaseName: name,
			Namespace:   "default",
		})
	})

	vals, err := suite.client.GetReleaseValues(ctx, name, "default", false)
	if err != nil {
		t.Fatalf("GetReleaseValues: %v", err)
	}
	if vals["custom"] != "value" {
		t.Errorf("custom value = %v, want %q", vals["custom"], "value")
	}
}

// TestIntegration_GetHistory verifies that revision history is recorded.
func TestIntegration_GetHistory(t *testing.T) {
	ctx := context.Background()
	const name = "hello-history"

	installHello(t, name)

	// Create a second revision.
	if _, err := suite.client.Upgrade(ctx, outbound.HelmUpgradeRequest{
		ReleaseName: name,
		Namespace:   "default",
		ChartName:   suite.chartPath,
		Wait:        true,
		Timeout:     60,
	}); err != nil {
		t.Fatalf("Upgrade: %v", err)
	}

	history, err := suite.client.GetHistory(ctx, name, "default", 10)
	if err != nil {
		t.Fatalf("GetHistory: %v", err)
	}
	if len(history) < 2 {
		t.Errorf("history length = %d, want ≥ 2", len(history))
	}
}

// TestIntegration_GetReleaseManifest verifies that the rendered manifest
// is non-empty after install.
func TestIntegration_GetReleaseManifest(t *testing.T) {
	ctx := context.Background()
	const name = "hello-manifest"

	installHello(t, name)

	manifest, err := suite.client.GetReleaseManifest(ctx, name, "default")
	if err != nil {
		t.Fatalf("GetReleaseManifest: %v", err)
	}
	if manifest == "" {
		t.Error("manifest is empty")
	}
}

// TestIntegration_TemplateChart verifies that chart templates can be rendered
// locally without a cluster connection.
func TestIntegration_TemplateChart(t *testing.T) {
	ctx := context.Background()

	rendered, err := suite.client.TemplateChart(ctx, outbound.TemplateRequest{
		ReleaseName: "preview",
		Namespace:   "default",
		ChartName:   suite.chartPath,
	})
	if err != nil {
		t.Fatalf("TemplateChart: %v", err)
	}
	if rendered == "" {
		t.Error("rendered output is empty")
	}
}

// TestIntegration_ShowChartValues verifies that default chart values are returned.
func TestIntegration_ShowChartValues(t *testing.T) {
	ctx := context.Background()

	// hello chart has no values.yaml so result should be an empty map, not an error.
	vals, err := suite.client.ShowChartValues(ctx, suite.chartPath, "", "")
	if err != nil {
		t.Fatalf("ShowChartValues: %v", err)
	}
	// We expect a non-nil map (even if empty).
	if vals == nil {
		t.Error("ShowChartValues returned nil map")
	}
}

// TestIntegration_LintChart verifies that the test chart passes linting.
func TestIntegration_LintChart(t *testing.T) {
	ctx := context.Background()

	result, err := suite.client.LintChart(ctx, []string{suite.chartPath}, nil)
	if err != nil {
		t.Fatalf("LintChart: %v", err)
	}
	if len(result.Errors) > 0 {
		t.Errorf("lint errors: %v", result.Errors)
	}
}

// TestIntegration_Uninstall verifies that a release can be removed and
// subsequently cannot be retrieved.
func TestIntegration_Uninstall(t *testing.T) {
	ctx := context.Background()
	const name = "hello-uninstall"

	// Install without the cleanup helper so we can uninstall manually.
	if _, err := suite.client.Install(ctx, outbound.HelmInstallRequest{
		ReleaseName: name,
		Namespace:   "default",
		ChartName:   suite.chartPath,
		Wait:        true,
		Timeout:     60,
	}); err != nil {
		t.Fatalf("Install: %v", err)
	}

	if err := suite.client.Uninstall(ctx, outbound.HelmUninstallRequest{
		ReleaseName: name,
		Namespace:   "default",
	}); err != nil {
		t.Fatalf("Uninstall: %v", err)
	}

	// GetRelease should now return an error (release not found).
	if _, err := suite.client.GetRelease(ctx, name, "default"); err == nil {
		t.Error("expected error after uninstall, got nil")
	}
}
