//go:build integration

package helmintegration_test

import (
	"context"
	"testing"

	"github.com/vinaycharlie01/sh-mcp-go/internal/ports/outbound"
)

func TestGetRelease(t *testing.T) {
	ctx := context.Background()
	install(t, "rel-get", nil)

	rel, err := helmClient.GetRelease(ctx, "rel-get", "default")
	if err != nil {
		t.Fatalf("GetRelease: %v", err)
	}
	if rel.Name != "rel-get" {
		t.Errorf("name = %q, want rel-get", rel.Name)
	}
	if rel.Info.Status.String() != "deployed" {
		t.Errorf("status = %s, want deployed", rel.Info.Status)
	}
}

func TestListReleases(t *testing.T) {
	ctx := context.Background()
	install(t, "rel-list", nil)

	list, err := helmClient.ListReleases(ctx, "default")
	if err != nil {
		t.Fatalf("ListReleases: %v", err)
	}

	found := false
	for _, r := range list {
		if r.Name == "rel-list" {
			found = true

			break
		}
	}
	if !found {
		t.Errorf("rel-list not found in %d releases", len(list))
	}
}

func TestListReleasesFiltered(t *testing.T) {
	ctx := context.Background()
	install(t, "rel-filtered", nil)

	list, err := helmClient.ListReleasesFiltered(ctx, outbound.HelmListRequest{
		Namespace: "default",
		StateMask: "deployed",
	})
	if err != nil {
		t.Fatalf("ListReleasesFiltered: %v", err)
	}

	found := false
	for _, r := range list {
		if r.Name == "rel-filtered" {
			found = true

			break
		}
	}
	if !found {
		t.Errorf("rel-filtered not found in %d releases", len(list))
	}
}

func TestGetHistory(t *testing.T) {
	ctx := context.Background()
	install(t, "rel-history", nil)

	if _, err := helmClient.Upgrade(ctx, outbound.HelmUpgradeRequest{
		ReleaseName: "rel-history",
		Namespace:   "default",
		ChartName:   chartPath,
		Wait:        true,
		Timeout:     60,
	}); err != nil {
		t.Fatalf("Upgrade: %v", err)
	}

	history, err := helmClient.GetHistory(ctx, "rel-history", "default", 10)
	if err != nil {
		t.Fatalf("GetHistory: %v", err)
	}
	if len(history) < 2 {
		t.Errorf("history length = %d, want ≥ 2", len(history))
	}
}

func TestGetReleaseRevision(t *testing.T) {
	ctx := context.Background()
	install(t, "rel-revision", nil)

	if _, err := helmClient.Upgrade(ctx, outbound.HelmUpgradeRequest{
		ReleaseName: "rel-revision",
		Namespace:   "default",
		ChartName:   chartPath,
		Wait:        true,
		Timeout:     60,
	}); err != nil {
		t.Fatalf("Upgrade: %v", err)
	}

	rev, err := helmClient.GetReleaseRevision(ctx, "rel-revision", "default", 1)
	if err != nil {
		t.Fatalf("GetReleaseRevision: %v", err)
	}
	if rev.Version != 1 {
		t.Errorf("revision = %d, want 1", rev.Version)
	}
}

func TestGetReleaseValues(t *testing.T) {
	ctx := context.Background()
	install(t, "rel-values", map[string]any{"custom": "value"})

	vals, err := helmClient.GetReleaseValues(ctx, "rel-values", "default", false)
	if err != nil {
		t.Fatalf("GetReleaseValues: %v", err)
	}
	if vals["custom"] != "value" {
		t.Errorf("custom = %v, want %q", vals["custom"], "value")
	}
}

func TestGetReleaseManifest(t *testing.T) {
	ctx := context.Background()
	install(t, "rel-manifest", nil)

	manifest, err := helmClient.GetReleaseManifest(ctx, "rel-manifest", "default")
	if err != nil {
		t.Fatalf("GetReleaseManifest: %v", err)
	}
	if manifest == "" {
		t.Error("manifest is empty")
	}
}

func TestGetReleaseNotes(t *testing.T) {
	ctx := context.Background()
	install(t, "rel-notes", nil)

	// hello chart has no NOTES.txt; verify no error (empty string is acceptable).
	_, err := helmClient.GetReleaseNotes(ctx, "rel-notes", "default")
	if err != nil {
		t.Fatalf("GetReleaseNotes: %v", err)
	}
}

func TestGetReleaseMetadata(t *testing.T) {
	ctx := context.Background()
	install(t, "rel-meta", nil)

	meta, err := helmClient.GetReleaseMetadata(ctx, "rel-meta", "default", 0)
	if err != nil {
		t.Fatalf("GetReleaseMetadata: %v", err)
	}
	if meta.Name != "rel-meta" {
		t.Errorf("name = %q, want rel-meta", meta.Name)
	}
	if meta.Namespace != "default" {
		t.Errorf("namespace = %q, want default", meta.Namespace)
	}
	if meta.Status != "deployed" {
		t.Errorf("status = %q, want deployed", meta.Status)
	}
}

func TestGetReleaseStatusWithResources(t *testing.T) {
	ctx := context.Background()
	install(t, "rel-status", nil)

	details, err := helmClient.GetReleaseStatusWithResources(ctx, "rel-status", "default", 0)
	if err != nil {
		t.Fatalf("GetReleaseStatusWithResources: %v", err)
	}
	if details.Status != "deployed" {
		t.Errorf("status = %q, want deployed", details.Status)
	}
	if details.ReleaseName != "rel-status" {
		t.Errorf("release name = %q, want rel-status", details.ReleaseName)
	}
}

func TestGetReleaseHooks(t *testing.T) {
	ctx := context.Background()
	install(t, "rel-hooks", nil)

	// hello chart has no hooks; verify non-error and non-nil slice.
	hooks, err := helmClient.GetReleaseHooks(ctx, "rel-hooks", "default")
	if err != nil {
		t.Fatalf("GetReleaseHooks: %v", err)
	}
	if hooks == nil {
		t.Error("GetReleaseHooks returned nil slice")
	}
}
