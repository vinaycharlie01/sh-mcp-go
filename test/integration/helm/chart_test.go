//go:build integration

package helmintegration_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/vinaycharlie01/sh-mcp-go/internal/ports/outbound"
)

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

func TestShowChartReadme(t *testing.T) {
	ctx := context.Background()

	// hello chart has no README; verify no error (empty string is acceptable).
	_, err := helmClient.ShowChartReadme(ctx, chartPath, "", "")
	if err != nil {
		t.Fatalf("ShowChartReadme: %v", err)
	}
}

func TestShowChartCRDs(t *testing.T) {
	ctx := context.Background()

	// hello chart has no CRDs; verify non-nil slice and no error.
	crds, err := helmClient.ShowChartCRDs(ctx, chartPath, "", "")
	if err != nil {
		t.Fatalf("ShowChartCRDs: %v", err)
	}
	if crds == nil {
		t.Error("ShowChartCRDs returned nil slice")
	}
}

func TestShowChart(t *testing.T) {
	ctx := context.Background()

	details, err := helmClient.ShowChart(ctx, chartPath, "", "")
	if err != nil {
		t.Fatalf("ShowChart: %v", err)
	}
	if details == nil {
		t.Fatal("ShowChart returned nil")
	}
	if details.Metadata == nil {
		t.Error("chart metadata is nil")
	}
}

func TestValidateChart(t *testing.T) {
	ctx := context.Background()

	if err := helmClient.ValidateChart(ctx, chartPath, "", ""); err != nil {
		t.Fatalf("ValidateChart: %v", err)
	}
}

func TestListChartDependencies(t *testing.T) {
	ctx := context.Background()

	// hello chart declares no dependencies; expect empty list, not error.
	deps, err := helmClient.ListChartDependencies(ctx, chartPath, "", "")
	if err != nil {
		t.Fatalf("ListChartDependencies: %v", err)
	}
	if deps == nil {
		t.Error("ListChartDependencies returned nil slice")
	}
}

func TestPackageChart(t *testing.T) {
	ctx := context.Background()

	dest, err := os.MkdirTemp("", "helm-pkg-*")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(dest) })

	pkgPath, err := helmClient.PackageChart(ctx, outbound.PackageRequest{
		ChartPath:   chartPath,
		Destination: dest,
	})
	if err != nil {
		t.Fatalf("PackageChart: %v", err)
	}
	if !strings.HasSuffix(pkgPath, ".tgz") {
		t.Errorf("package path = %q, want *.tgz suffix", pkgPath)
	}
	if _, err = os.Stat(pkgPath); err != nil {
		t.Errorf("packaged file not found: %v", err)
	}
}
