package sqlite_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/vinaycharlie01/sh-mcp-go/internal/adapters/storage/sqlite"
	"github.com/vinaycharlie01/sh-mcp-go/internal/domain/deployment"
)

func newTestRepo(t *testing.T) (*sqlite.Repository, func()) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")

	repo, err := sqlite.NewRepository(context.Background(), path)
	if err != nil {
		t.Fatalf("NewRepository: %v", err)
	}

	return repo, func() {
		_ = repo.Close()
		_ = os.Remove(path)
	}
}

func newDeployment(t *testing.T, release, namespace string) *deployment.Deployment {
	t.Helper()
	d, err := deployment.New(
		deployment.ReleaseName(release),
		deployment.Namespace(namespace),
		deployment.ChartReference{
			Name:    "prometheus",
			RepoURL: "https://prometheus-community.github.io/helm-charts",
			Version: "25.0.0",
			Source:  deployment.ChartSourceRepo,
		},
		deployment.Values{"replicaCount": 1},
	)
	if err != nil {
		t.Fatalf("creating deployment: %v", err)
	}

	return d
}

func TestRepository_SaveAndFindByID(t *testing.T) {
	repo, cleanup := newTestRepo(t)
	defer cleanup()

	ctx := context.Background()
	d := newDeployment(t, "prometheus", "monitoring")

	if err := repo.Save(ctx, d); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := repo.FindByID(ctx, d.ID())
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}

	if got.ID() != d.ID() {
		t.Errorf("ID mismatch: got %s, want %s", got.ID(), d.ID())
	}
	if got.Name() != d.Name() {
		t.Errorf("Name mismatch: got %s, want %s", got.Name(), d.Name())
	}
	if got.Status() != d.Status() {
		t.Errorf("Status mismatch: got %s, want %s", got.Status(), d.Status())
	}
}

func TestRepository_FindByReleaseName(t *testing.T) {
	repo, cleanup := newTestRepo(t)
	defer cleanup()

	ctx := context.Background()
	d := newDeployment(t, "grafana", "monitoring")
	_ = repo.Save(ctx, d)

	got, err := repo.FindByReleaseName(ctx, deployment.ReleaseName("grafana"), deployment.Namespace("monitoring"))
	if err != nil {
		t.Fatalf("FindByReleaseName: %v", err)
	}
	if got.ID() != d.ID() {
		t.Errorf("ID mismatch")
	}
}

func TestRepository_Save_UpdatesExistingRecord(t *testing.T) {
	repo, cleanup := newTestRepo(t)
	defer cleanup()

	ctx := context.Background()
	d := newDeployment(t, "redis", "data")
	_ = repo.Save(ctx, d)

	_ = d.StartDeployment()
	_ = d.MarkSucceeded(1)
	if err := repo.Save(ctx, d); err != nil {
		t.Fatalf("Save update: %v", err)
	}

	got, err := repo.FindByID(ctx, d.ID())
	if err != nil {
		t.Fatalf("FindByID after update: %v", err)
	}
	if got.Status() != deployment.StatusSucceeded {
		t.Errorf("expected SUCCEEDED, got %s", got.Status())
	}
}

func TestRepository_ListByNamespace(t *testing.T) {
	repo, cleanup := newTestRepo(t)
	defer cleanup()

	ctx := context.Background()
	_ = repo.Save(ctx, newDeployment(t, "app1", "prod"))
	_ = repo.Save(ctx, newDeployment(t, "app2", "prod"))
	_ = repo.Save(ctx, newDeployment(t, "app3", "staging"))

	results, err := repo.ListByNamespace(ctx, deployment.Namespace("prod"))
	if err != nil {
		t.Fatalf("ListByNamespace: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2, got %d", len(results))
	}
}

func TestRepository_Delete(t *testing.T) {
	repo, cleanup := newTestRepo(t)
	defer cleanup()

	ctx := context.Background()
	d := newDeployment(t, "temp-app", "default")
	_ = repo.Save(ctx, d)
	_ = repo.Delete(ctx, d.ID())

	_, err := repo.FindByID(ctx, d.ID())
	if err == nil {
		t.Error("expected error after deletion")
	}
}
