//go:build integration

package helmintegration_test

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

var (
	helmClient *helmadapter.Client
	chartPath  string
)

// TestMain starts a k3s testcontainer, wires up the Helm client, then runs all tests.
func TestMain(m *testing.M) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	container, err := k3s.Run(ctx, "rancher/k3s:v1.31.4-k3s1")
	if err != nil {
		panic("k3s.Run: " + err.Error())
	}
	defer func() { _ = container.Terminate(context.Background()) }()

	kubeconfig, err := container.GetKubeConfig(ctx)
	if err != nil {
		panic("GetKubeConfig: " + err.Error())
	}

	f, err := os.CreateTemp("", "kubeconfig-*.yaml")
	if err != nil {
		panic("CreateTemp: " + err.Error())
	}
	_, _ = f.Write(kubeconfig)
	f.Close()
	defer os.Remove(f.Name())
	os.Setenv("KUBECONFIG", f.Name())

	dir, err := os.MkdirTemp("", "helm-*")
	if err != nil {
		panic("MkdirTemp: " + err.Error())
	}
	defer os.RemoveAll(dir)

	helmClient, err = helmadapter.NewClient(&config.HelmConfig{
		RepositoryCache:  filepath.Join(dir, "cache"),
		RepositoryConfig: filepath.Join(dir, "repositories.yaml"),
		DefaultTimeout:   2 * time.Minute,
	})
	if err != nil {
		panic("NewClient: " + err.Error())
	}

	wd, _ := os.Getwd()
	chartPath = filepath.Join(wd, "testdata", "charts", "hello")

	os.Exit(m.Run())
}

// install is a test helper that installs the hello chart and registers cleanup.
func install(t *testing.T, release string, values map[string]any) {
	t.Helper()
	ctx := context.Background()

	_, err := helmClient.Install(ctx, outbound.HelmInstallRequest{
		ReleaseName: release,
		Namespace:   "default",
		ChartName:   chartPath,
		Values:      values,
		Wait:        true,
		Timeout:     60,
	})
	if err != nil {
		t.Fatalf("Install(%q): %v", release, err)
	}

	t.Cleanup(func() {
		_ = helmClient.Uninstall(context.Background(), outbound.HelmUninstallRequest{
			ReleaseName: release,
			Namespace:   "default",
		})
	})
}

func TestInstall(t *testing.T) {
	ctx := context.Background()
	install(t, "hello-install", nil)

	rel, err := helmClient.GetRelease(ctx, "hello-install", "default")
	if err != nil {
		t.Fatalf("GetRelease: %v", err)
	}
	if rel.Name != "hello-install" {
		t.Errorf("name = %q, want %q", rel.Name, "hello-install")
	}
	if rel.Info.Status.String() != "deployed" {
		t.Errorf("status = %s, want deployed", rel.Info.Status)
	}
}

func TestListReleases(t *testing.T) {
	ctx := context.Background()
	install(t, "hello-list", nil)

	list, err := helmClient.ListReleases(ctx, "default")
	if err != nil {
		t.Fatalf("ListReleases: %v", err)
	}

	found := false
	for _, r := range list {
		if r.Name == "hello-list" {
			found = true

			break
		}
	}
	if !found {
		t.Errorf("hello-list not found in list (%d releases)", len(list))
	}
}

func TestGetReleaseValues(t *testing.T) {
	ctx := context.Background()
	install(t, "hello-values", map[string]any{"custom": "value"})

	vals, err := helmClient.GetReleaseValues(ctx, "hello-values", "default", false)
	if err != nil {
		t.Fatalf("GetReleaseValues: %v", err)
	}
	if vals["custom"] != "value" {
		t.Errorf("custom = %v, want %q", vals["custom"], "value")
	}
}

func TestGetReleaseManifest(t *testing.T) {
	ctx := context.Background()
	install(t, "hello-manifest", nil)

	manifest, err := helmClient.GetReleaseManifest(ctx, "hello-manifest", "default")
	if err != nil {
		t.Fatalf("GetReleaseManifest: %v", err)
	}
	if manifest == "" {
		t.Error("manifest is empty")
	}
}

func TestUpgrade(t *testing.T) {
	ctx := context.Background()
	install(t, "hello-upgrade", nil)

	rel, err := helmClient.Upgrade(ctx, outbound.HelmUpgradeRequest{
		ReleaseName: "hello-upgrade",
		Namespace:   "default",
		ChartName:   chartPath,
		Values:      map[string]any{"greeting": "hi"},
		Wait:        true,
		Timeout:     60,
	})
	if err != nil {
		t.Fatalf("Upgrade: %v", err)
	}
	if rel.Version != 2 {
		t.Errorf("revision = %d, want 2", rel.Version)
	}
}

func TestGetHistory(t *testing.T) {
	ctx := context.Background()
	install(t, "hello-history", nil)

	if _, err := helmClient.Upgrade(ctx, outbound.HelmUpgradeRequest{
		ReleaseName: "hello-history",
		Namespace:   "default",
		ChartName:   chartPath,
		Wait:        true,
		Timeout:     60,
	}); err != nil {
		t.Fatalf("Upgrade: %v", err)
	}

	history, err := helmClient.GetHistory(ctx, "hello-history", "default", 10)
	if err != nil {
		t.Fatalf("GetHistory: %v", err)
	}
	if len(history) < 2 {
		t.Errorf("history length = %d, want ≥ 2", len(history))
	}
}

func TestRollback(t *testing.T) {
	ctx := context.Background()
	install(t, "hello-rollback", nil)

	if _, err := helmClient.Upgrade(ctx, outbound.HelmUpgradeRequest{
		ReleaseName: "hello-rollback",
		Namespace:   "default",
		ChartName:   chartPath,
		Wait:        true,
		Timeout:     60,
	}); err != nil {
		t.Fatalf("Upgrade (pre-rollback): %v", err)
	}

	if err := helmClient.Rollback(ctx, outbound.HelmRollbackRequest{
		ReleaseName: "hello-rollback",
		Namespace:   "default",
		Version:     1,
		Wait:        true,
		Timeout:     60,
	}); err != nil {
		t.Fatalf("Rollback: %v", err)
	}

	rel, err := helmClient.GetRelease(ctx, "hello-rollback", "default")
	if err != nil {
		t.Fatalf("GetRelease after rollback: %v", err)
	}
	if rel.Info.Status.String() != "deployed" {
		t.Errorf("status = %s, want deployed", rel.Info.Status)
	}
}

func TestUninstall(t *testing.T) {
	ctx := context.Background()

	if _, err := helmClient.Install(ctx, outbound.HelmInstallRequest{
		ReleaseName: "hello-uninstall",
		Namespace:   "default",
		ChartName:   chartPath,
		Wait:        true,
		Timeout:     60,
	}); err != nil {
		t.Fatalf("Install: %v", err)
	}

	if err := helmClient.Uninstall(ctx, outbound.HelmUninstallRequest{
		ReleaseName: "hello-uninstall",
		Namespace:   "default",
	}); err != nil {
		t.Fatalf("Uninstall: %v", err)
	}

	if _, err := helmClient.GetRelease(ctx, "hello-uninstall", "default"); err == nil {
		t.Error("expected error after uninstall, got nil")
	}
}

func TestTemplateChart(t *testing.T) {
	ctx := context.Background()

	rendered, err := helmClient.TemplateChart(ctx, outbound.TemplateRequest{
		ReleaseName: "preview",
		Namespace:   "default",
		ChartName:   chartPath,
	})
	if err != nil {
		t.Fatalf("TemplateChart: %v", err)
	}
	if rendered == "" {
		t.Error("rendered output is empty")
	}
}

func TestShowChartValues(t *testing.T) {
	ctx := context.Background()

	vals, err := helmClient.ShowChartValues(ctx, chartPath, "", "")
	if err != nil {
		t.Fatalf("ShowChartValues: %v", err)
	}
	if vals == nil {
		t.Error("ShowChartValues returned nil map")
	}
}

func TestLintChart(t *testing.T) {
	ctx := context.Background()

	result, err := helmClient.LintChart(ctx, []string{chartPath}, nil)
	if err != nil {
		t.Fatalf("LintChart: %v", err)
	}
	if len(result.Errors) > 0 {
		t.Errorf("lint errors: %v", result.Errors)
	}
}
