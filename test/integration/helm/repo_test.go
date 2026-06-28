//go:build integration

package helmintegration_test

import (
	"context"
	"testing"

	"github.com/vinaycharlie01/sh-mcp-go/internal/ports/outbound"
)

const (
	testRepoName = "bitnami"
	testRepoURL  = "https://charts.bitnami.com/bitnami"
)

func TestAddRepo(t *testing.T) {
	ctx := context.Background()

	if err := helmClient.AddRepo(ctx, outbound.RepoEntry{
		Name: testRepoName,
		URL:  testRepoURL,
	}); err != nil {
		t.Fatalf("AddRepo: %v", err)
	}
	t.Cleanup(func() {
		_ = helmClient.RemoveRepo(ctx, testRepoName)
	})

	repos, err := helmClient.ListRepos(ctx)
	if err != nil {
		t.Fatalf("ListRepos after add: %v", err)
	}

	found := false
	for _, r := range repos {
		if r.Name == testRepoName {
			found = true

			break
		}
	}
	if !found {
		t.Errorf("%q not found after AddRepo (%d repos)", testRepoName, len(repos))
	}
}

func TestListRepos(t *testing.T) {
	ctx := context.Background()

	if err := helmClient.AddRepo(ctx, outbound.RepoEntry{
		Name: testRepoName,
		URL:  testRepoURL,
	}); err != nil {
		t.Fatalf("AddRepo: %v", err)
	}
	t.Cleanup(func() { _ = helmClient.RemoveRepo(ctx, testRepoName) })

	repos, err := helmClient.ListRepos(ctx)
	if err != nil {
		t.Fatalf("ListRepos: %v", err)
	}
	if len(repos) == 0 {
		t.Error("expected at least one repo, got none")
	}
}

func TestSearchRepo(t *testing.T) {
	ctx := context.Background()

	if err := helmClient.AddRepo(ctx, outbound.RepoEntry{
		Name: testRepoName,
		URL:  testRepoURL,
	}); err != nil {
		t.Fatalf("AddRepo: %v", err)
	}
	t.Cleanup(func() { _ = helmClient.RemoveRepo(ctx, testRepoName) })

	results, err := helmClient.SearchRepo(ctx, "nginx", "")
	if err != nil {
		t.Fatalf("SearchRepo: %v", err)
	}
	if len(results) == 0 {
		t.Error("expected search results for nginx, got none")
	}
}

func TestUpdateRepo(t *testing.T) {
	ctx := context.Background()

	if err := helmClient.AddRepo(ctx, outbound.RepoEntry{
		Name: testRepoName,
		URL:  testRepoURL,
	}); err != nil {
		t.Fatalf("AddRepo: %v", err)
	}
	t.Cleanup(func() { _ = helmClient.RemoveRepo(ctx, testRepoName) })

	if err := helmClient.UpdateRepo(ctx, testRepoName); err != nil {
		t.Fatalf("UpdateRepo: %v", err)
	}
}

func TestRemoveRepo(t *testing.T) {
	ctx := context.Background()

	if err := helmClient.AddRepo(ctx, outbound.RepoEntry{
		Name: testRepoName,
		URL:  testRepoURL,
	}); err != nil {
		t.Fatalf("AddRepo: %v", err)
	}

	if err := helmClient.RemoveRepo(ctx, testRepoName); err != nil {
		t.Fatalf("RemoveRepo: %v", err)
	}

	repos, err := helmClient.ListRepos(ctx)
	if err != nil {
		t.Fatalf("ListRepos after remove: %v", err)
	}
	for _, r := range repos {
		if r.Name == testRepoName {
			t.Errorf("%q still present after RemoveRepo", testRepoName)
		}
	}
}
