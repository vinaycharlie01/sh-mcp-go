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

// install is a shared helper that installs the hello chart and registers cleanup.
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
