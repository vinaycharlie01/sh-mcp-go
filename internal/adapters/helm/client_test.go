package helm

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	repov1 "helm.sh/helm/v4/pkg/repo/v1"

	"github.com/vinaycharlie01/sh-mcp-go/internal/infrastructure/config"
	"github.com/vinaycharlie01/sh-mcp-go/internal/ports/outbound"
)

// compile-time interface compliance check.
var _ outbound.HelmPort = (*Client)(nil)

func newTestClient(t *testing.T) *Client {
	t.Helper()

	dir := t.TempDir()
	cfg := &config.HelmConfig{
		RepositoryCache:  filepath.Join(dir, "cache"),
		RepositoryConfig: filepath.Join(dir, "repositories.yaml"),
	}

	c, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	return c
}

func TestNewClient_CreatesDirectories(t *testing.T) {
	c := newTestClient(t)

	if _, err := os.Stat(c.cfg.RepositoryCache); err != nil {
		t.Errorf("cache dir not created: %v", err)
	}
}

func TestListRepos_EmptyWhenNoConfig(t *testing.T) {
	c := newTestClient(t)

	repos, err := c.ListRepos(context.Background())
	if err != nil {
		t.Fatalf("ListRepos: %v", err)
	}

	if len(repos) != 0 {
		t.Errorf("expected 0 repos, got %d", len(repos))
	}
}

// writeRepoEntry writes an entry directly to the repos file without downloading the index.
// This is used in tests to avoid network calls.
func writeRepoEntry(t *testing.T, c *Client, entry outbound.RepoEntry) {
	t.Helper()

	f, err := repov1.LoadFile(c.settings.RepositoryConfig)
	if err != nil {
		f = repov1.NewFile()
	}

	f.Update(&repov1.Entry{
		Name:                  entry.Name,
		URL:                   entry.URL,
		Username:              entry.Username,
		Password:              entry.Password,
		CAFile:                entry.CAFile,
		CertFile:              entry.CertFile,
		KeyFile:               entry.KeyFile,
		InsecureSkipTLSVerify: entry.InsecureSkipTLSVerify,
		PassCredentialsAll:    entry.PassCredentialsAll,
	})

	if err := f.WriteFile(c.settings.RepositoryConfig, filePerm); err != nil {
		t.Fatalf("writing repo file: %v", err)
	}
}

func TestListRepos_ReturnsEntries(t *testing.T) {
	c := newTestClient(t)

	writeRepoEntry(t, c, outbound.RepoEntry{Name: "stable", URL: "https://charts.helm.sh/stable"})
	writeRepoEntry(t, c, outbound.RepoEntry{Name: "bitnami", URL: "https://charts.bitnami.com/bitnami"})

	repos, err := c.ListRepos(context.Background())
	if err != nil {
		t.Fatalf("ListRepos: %v", err)
	}

	if len(repos) != 2 {
		t.Errorf("expected 2 repos, got %d", len(repos))
	}
}

func TestRemoveRepo_RemovesEntry(t *testing.T) {
	c := newTestClient(t)
	ctx := context.Background()

	writeRepoEntry(t, c, outbound.RepoEntry{Name: "to-remove", URL: "https://charts.example.com"})

	if err := c.RemoveRepo(ctx, "to-remove"); err != nil {
		t.Fatalf("RemoveRepo: %v", err)
	}

	repos, err := c.ListRepos(ctx)
	if err != nil {
		t.Fatalf("ListRepos after remove: %v", err)
	}

	if len(repos) != 0 {
		t.Errorf("expected 0 repos after remove, got %d", len(repos))
	}
}

func TestRemoveRepo_NotFound(t *testing.T) {
	c := newTestClient(t)

	err := c.RemoveRepo(context.Background(), "nonexistent")
	if err == nil {
		t.Error("expected error removing nonexistent repo")
	}
}

func TestSearchRepo_MatchesByName(t *testing.T) {
	c := newTestClient(t)

	if err := os.MkdirAll(c.cfg.RepositoryCache, 0o755); err != nil {
		t.Fatal(err)
	}

	idxContent := `apiVersion: v1
entries:
  nginx:
    - name: nginx
      version: "1.0.0"
      appVersion: "1.25.0"
      description: "NGINX chart"
  redis:
    - name: redis
      version: "2.0.0"
      appVersion: "7.0.0"
      description: "Redis chart"
generated: "2024-01-01T00:00:00Z"
`
	if err := os.WriteFile(filepath.Join(c.cfg.RepositoryCache, "test-index.yaml"), []byte(idxContent), 0o644); err != nil {
		t.Fatal(err)
	}

	results, err := c.SearchRepo(context.Background(), "nginx", "")
	if err != nil {
		t.Fatalf("SearchRepo: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	if results[0].Name != "nginx" {
		t.Errorf("expected nginx, got %s", results[0].Name)
	}

	if results[0].ChartVersion != "1.0.0" {
		t.Errorf("expected version 1.0.0, got %s", results[0].ChartVersion)
	}
}

func TestSearchRepo_EmptyKeyword_ReturnsAll(t *testing.T) {
	c := newTestClient(t)

	if err := os.MkdirAll(c.cfg.RepositoryCache, 0o755); err != nil {
		t.Fatal(err)
	}

	idxContent := `apiVersion: v1
entries:
  alpha:
    - name: alpha
      version: "1.0.0"
  beta:
    - name: beta
      version: "1.0.0"
generated: "2024-01-01T00:00:00Z"
`
	if err := os.WriteFile(filepath.Join(c.cfg.RepositoryCache, "all-index.yaml"), []byte(idxContent), 0o644); err != nil {
		t.Fatal(err)
	}

	results, err := c.SearchRepo(context.Background(), "", "")
	if err != nil {
		t.Fatalf("SearchRepo empty keyword: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}
}

func TestRepoEntry_TLSAndAuthFields(t *testing.T) {
	c := newTestClient(t)

	entry := outbound.RepoEntry{
		Name:                  "secure-repo",
		URL:                   "https://secure.example.com",
		Username:              "user",
		Password:              "pass",
		CAFile:                "/etc/ssl/ca.crt",
		CertFile:              "/etc/ssl/client.crt",
		KeyFile:               "/etc/ssl/client.key",
		InsecureSkipTLSVerify: false,
		PassCredentialsAll:    true,
	}

	writeRepoEntry(t, c, entry)

	repos, err := c.ListRepos(context.Background())
	if err != nil {
		t.Fatalf("ListRepos: %v", err)
	}

	if len(repos) != 1 {
		t.Fatalf("expected 1 repo, got %d", len(repos))
	}

	got := repos[0]
	if got.Username != entry.Username {
		t.Errorf("username mismatch: got %q, want %q", got.Username, entry.Username)
	}

	if got.CAFile != entry.CAFile {
		t.Errorf("ca_file mismatch: got %q, want %q", got.CAFile, entry.CAFile)
	}

	if !got.PassCredentialsAll {
		t.Error("expected PassCredentialsAll=true")
	}
}
