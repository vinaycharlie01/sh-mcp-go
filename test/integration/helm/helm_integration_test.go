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

// TestHelm_Lifecycle exercises the full release lifecycle on a single release:
// install → inspect → upgrade → history → rollback → uninstall.
// Subtests run sequentially; each one depends on the previous step succeeding.
func TestHelm_Lifecycle(t *testing.T) {
	ctx := context.Background()
	const (
		release   = "hello"
		namespace = "default"
	)

	t.Run("Install", func(t *testing.T) {
		rel, err := helmClient.Install(ctx, outbound.HelmInstallRequest{
			ReleaseName: release,
			Namespace:   namespace,
			ChartName:   chartPath,
			Values:      map[string]any{"custom": "value"},
			Wait:        true,
			Timeout:     60,
		})
		if err != nil {
			t.Fatalf("Install: %v", err)
		}
		if rel.Info.Status.String() != "deployed" {
			t.Fatalf("status = %s, want deployed", rel.Info.Status)
		}
	})

	t.Run("Get", func(t *testing.T) {
		rel, err := helmClient.GetRelease(ctx, release, namespace)
		if err != nil {
			t.Fatalf("GetRelease: %v", err)
		}
		if rel.Name != release {
			t.Errorf("name = %q, want %q", rel.Name, release)
		}
	})

	t.Run("List", func(t *testing.T) {
		list, err := helmClient.ListReleases(ctx, namespace)
		if err != nil {
			t.Fatalf("ListReleases: %v", err)
		}
		found := false
		for _, r := range list {
			if r.Name == release {
				found = true

				break
			}
		}
		if !found {
			t.Errorf("%q not found in list (%d releases)", release, len(list))
		}
	})

	t.Run("Values", func(t *testing.T) {
		vals, err := helmClient.GetReleaseValues(ctx, release, namespace, false)
		if err != nil {
			t.Fatalf("GetReleaseValues: %v", err)
		}
		if vals["custom"] != "value" {
			t.Errorf("custom = %v, want %q", vals["custom"], "value")
		}
	})

	t.Run("Manifest", func(t *testing.T) {
		manifest, err := helmClient.GetReleaseManifest(ctx, release, namespace)
		if err != nil {
			t.Fatalf("GetReleaseManifest: %v", err)
		}
		if manifest == "" {
			t.Error("manifest is empty")
		}
	})

	t.Run("Upgrade", func(t *testing.T) {
		rel, err := helmClient.Upgrade(ctx, outbound.HelmUpgradeRequest{
			ReleaseName: release,
			Namespace:   namespace,
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
	})

	t.Run("History", func(t *testing.T) {
		history, err := helmClient.GetHistory(ctx, release, namespace, 10)
		if err != nil {
			t.Fatalf("GetHistory: %v", err)
		}
		if len(history) < 2 {
			t.Errorf("history length = %d, want ≥ 2", len(history))
		}
	})

	t.Run("Rollback", func(t *testing.T) {
		if err := helmClient.Rollback(ctx, outbound.HelmRollbackRequest{
			ReleaseName: release,
			Namespace:   namespace,
			Version:     1,
			Wait:        true,
			Timeout:     60,
		}); err != nil {
			t.Fatalf("Rollback: %v", err)
		}
		rel, err := helmClient.GetRelease(ctx, release, namespace)
		if err != nil {
			t.Fatalf("GetRelease after rollback: %v", err)
		}
		if rel.Info.Status.String() != "deployed" {
			t.Errorf("status = %s, want deployed", rel.Info.Status)
		}
	})

	t.Run("Uninstall", func(t *testing.T) {
		if err := helmClient.Uninstall(ctx, outbound.HelmUninstallRequest{
			ReleaseName: release,
			Namespace:   namespace,
		}); err != nil {
			t.Fatalf("Uninstall: %v", err)
		}
		if _, err := helmClient.GetRelease(ctx, release, namespace); err == nil {
			t.Error("expected error after uninstall, got nil")
		}
	})
}

// TestHelm_LocalOps covers chart operations that work without cluster state.
func TestHelm_LocalOps(t *testing.T) {
	ctx := context.Background()

	t.Run("Template", func(t *testing.T) {
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
	})

	t.Run("ShowValues", func(t *testing.T) {
		vals, err := helmClient.ShowChartValues(ctx, chartPath, "", "")
		if err != nil {
			t.Fatalf("ShowChartValues: %v", err)
		}
		if vals == nil {
			t.Error("ShowChartValues returned nil map")
		}
	})

	t.Run("Lint", func(t *testing.T) {
		result, err := helmClient.LintChart(ctx, []string{chartPath}, nil)
		if err != nil {
			t.Fatalf("LintChart: %v", err)
		}
		if len(result.Errors) > 0 {
			t.Errorf("lint errors: %v", result.Errors)
		}
	})
}
