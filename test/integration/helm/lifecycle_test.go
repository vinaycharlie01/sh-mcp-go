//go:build integration

package helmintegration_test

import (
	"context"
	"testing"

	"github.com/vinaycharlie01/sh-mcp-go/internal/ports/outbound"
)

func TestInstall(t *testing.T) {
	ctx := context.Background()
	install(t, "lc-install", nil)

	rel, err := helmClient.GetRelease(ctx, "lc-install", "default")
	if err != nil {
		t.Fatalf("GetRelease: %v", err)
	}
	if rel.Info.Status.String() != "deployed" {
		t.Errorf("status = %s, want deployed", rel.Info.Status)
	}
}

func TestDryRunInstall(t *testing.T) {
	ctx := context.Background()

	manifest, err := helmClient.DryRunInstall(ctx, outbound.HelmInstallRequest{
		ReleaseName: "lc-dryrun",
		Namespace:   "default",
		ChartName:   chartPath,
		Timeout:     60,
	})
	if err != nil {
		t.Fatalf("DryRunInstall: %v", err)
	}
	if manifest == "" {
		t.Error("dry-run manifest is empty")
	}

	// Dry run must not leave a real release behind.
	if _, err = helmClient.GetRelease(ctx, "lc-dryrun", "default"); err == nil {
		t.Error("dry-run left a real release behind")
	}
}

func TestUpgrade(t *testing.T) {
	ctx := context.Background()
	install(t, "lc-upgrade", nil)

	rel, err := helmClient.Upgrade(ctx, outbound.HelmUpgradeRequest{
		ReleaseName: "lc-upgrade",
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

func TestDryRunUpgrade(t *testing.T) {
	ctx := context.Background()
	install(t, "lc-dryupgrade", nil)

	manifest, err := helmClient.DryRunUpgrade(ctx, outbound.HelmUpgradeRequest{
		ReleaseName: "lc-dryupgrade",
		Namespace:   "default",
		ChartName:   chartPath,
		Values:      map[string]any{"greeting": "hello"},
		Timeout:     60,
	})
	if err != nil {
		t.Fatalf("DryRunUpgrade: %v", err)
	}
	if manifest == "" {
		t.Error("dry-run upgrade manifest is empty")
	}
}

func TestDiff(t *testing.T) {
	ctx := context.Background()
	install(t, "lc-diff", nil)

	result, err := helmClient.Diff(ctx, outbound.HelmUpgradeRequest{
		ReleaseName: "lc-diff",
		Namespace:   "default",
		ChartName:   chartPath,
		Values:      map[string]any{"greeting": "changed"},
		Timeout:     60,
	})
	if err != nil {
		t.Fatalf("Diff: %v", err)
	}
	if result == nil {
		t.Fatal("Diff returned nil result")
	}
}

func TestRollback(t *testing.T) {
	ctx := context.Background()
	install(t, "lc-rollback", nil)

	if _, err := helmClient.Upgrade(ctx, outbound.HelmUpgradeRequest{
		ReleaseName: "lc-rollback",
		Namespace:   "default",
		ChartName:   chartPath,
		Wait:        true,
		Timeout:     60,
	}); err != nil {
		t.Fatalf("Upgrade (pre-rollback): %v", err)
	}

	if err := helmClient.Rollback(ctx, outbound.HelmRollbackRequest{
		ReleaseName: "lc-rollback",
		Namespace:   "default",
		Version:     1,
		Wait:        true,
		Timeout:     60,
	}); err != nil {
		t.Fatalf("Rollback: %v", err)
	}

	rel, err := helmClient.GetRelease(ctx, "lc-rollback", "default")
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
		ReleaseName: "lc-uninstall",
		Namespace:   "default",
		ChartName:   chartPath,
		Wait:        true,
		Timeout:     60,
	}); err != nil {
		t.Fatalf("Install: %v", err)
	}

	if err := helmClient.Uninstall(ctx, outbound.HelmUninstallRequest{
		ReleaseName: "lc-uninstall",
		Namespace:   "default",
	}); err != nil {
		t.Fatalf("Uninstall: %v", err)
	}

	if _, err := helmClient.GetRelease(ctx, "lc-uninstall", "default"); err == nil {
		t.Error("expected error after uninstall, got nil")
	}
}
