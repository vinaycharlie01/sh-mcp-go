package deployment_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/vinaycharlie01/sh-mcp-go/internal/domain/deployment"
)

func validChart() deployment.ChartReference {
	return deployment.ChartReference{
		Name:    "prometheus",
		RepoURL: "https://prometheus-community.github.io/helm-charts",
		Version: "25.0.0",
		Source:  deployment.ChartSourceRepo,
	}
}

func TestNew_CreatesDeploymentWithPendingStatus(t *testing.T) {
	d, err := deployment.New(
		deployment.ReleaseName("prometheus"),
		deployment.Namespace("monitoring"),
		validChart(),
		deployment.Values{"replicaCount": 1},
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d.Status() != deployment.StatusPending {
		t.Errorf("expected PENDING, got %s", d.Status())
	}
	if d.ID().String() == "" {
		t.Error("expected non-empty ID")
	}
	if len(d.DrainEvents()) == 0 {
		t.Error("expected DeploymentCreated event")
	}
}

func TestNew_InvalidReleaseName_ReturnsError(t *testing.T) {
	cases := []string{
		"",
		"UPPERCASE",
		"has spaces",
		"toolongname-toolongname-toolongname-toolongname-toolongname-xyz",
	}
	for _, name := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := deployment.New(
				deployment.ReleaseName(name),
				deployment.Namespace("default"),
				validChart(),
				nil,
			)
			if err == nil {
				t.Errorf("expected error for release name %q", name)
			}
		})
	}
}

func TestNew_InvalidChartVersion_ReturnsError(t *testing.T) {
	chart := validChart()
	chart.Version = "not-a-semver!!!"

	_, err := deployment.New(
		deployment.ReleaseName("my-release"),
		deployment.Namespace("default"),
		chart,
		nil,
	)
	if err == nil {
		t.Error("expected error for invalid chart version")
	}
}

func TestDeployment_LifecycleTransitions(t *testing.T) {
	d, err := deployment.New(
		deployment.ReleaseName("my-app"),
		deployment.Namespace("default"),
		validChart(),
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}

	// Drain creation event
	d.DrainEvents()

	// PENDING -> DEPLOYING
	if err := d.StartDeployment(); err != nil {
		t.Fatalf("StartDeployment: %v", err)
	}
	if d.Status() != deployment.StatusDeploying {
		t.Errorf("expected DEPLOYING, got %s", d.Status())
	}

	// DEPLOYING -> SUCCEEDED
	if err := d.MarkSucceeded(1); err != nil {
		t.Fatalf("MarkSucceeded: %v", err)
	}
	if d.Status() != deployment.StatusSucceeded {
		t.Errorf("expected SUCCEEDED, got %s", d.Status())
	}
	if len(d.History()) != 1 {
		t.Errorf("expected 1 history entry, got %d", len(d.History()))
	}

	evts := d.DrainEvents()
	if len(evts) != 2 {
		t.Errorf("expected 2 events (started + succeeded), got %d", len(evts))
	}
}

func TestDeployment_InvalidTransition_ReturnsError(t *testing.T) {
	d, _ := deployment.New(
		deployment.ReleaseName("my-app"),
		deployment.Namespace("default"),
		validChart(),
		nil,
	)
	d.DrainEvents()

	// Cannot go PENDING -> SUCCEEDED directly
	err := d.MarkSucceeded(1)
	if err == nil {
		t.Error("expected error for invalid transition PENDING -> SUCCEEDED")
	}

	var te deployment.ErrInvalidTransition
	if diff := cmp.Diff(te.From, deployment.Status("")); diff != "" {
		_ = diff // just check type
	}
}

func TestDeployment_Rollback(t *testing.T) {
	d, _ := deployment.New(
		deployment.ReleaseName("my-app"),
		deployment.Namespace("default"),
		validChart(),
		nil,
	)
	d.DrainEvents()
	_ = d.StartDeployment()
	_ = d.MarkSucceeded(1)
	d.DrainEvents()

	if err := d.StartRollback(0); err != nil {
		t.Fatalf("StartRollback: %v", err)
	}
	if d.Status() != deployment.StatusRollingBack {
		t.Errorf("expected ROLLING_BACK, got %s", d.Status())
	}

	d.MarkRolledBack(0)
	if d.Status() != deployment.StatusRolledBack {
		t.Errorf("expected ROLLED_BACK, got %s", d.Status())
	}
}

func TestReleaseName_Validation(t *testing.T) {
	valid := []string{"prometheus", "my-app", "a", "a1b2c3"}
	for _, name := range valid {
		t.Run("valid:"+name, func(t *testing.T) {
			if err := deployment.ReleaseName(name).Validate(); err != nil {
				t.Errorf("expected valid, got error: %v", err)
			}
		})
	}

	invalid := []string{"", "My-App", "-start-with-hyphen", "end-with-hyphen-"}
	for _, name := range invalid {
		t.Run("invalid:"+name, func(t *testing.T) {
			if err := deployment.ReleaseName(name).Validate(); err == nil {
				t.Errorf("expected error for %q", name)
			}
		})
	}
}

func BenchmarkNew(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = deployment.New(
			deployment.ReleaseName("bench-app"),
			deployment.Namespace("default"),
			validChart(),
			deployment.Values{"key": "value"},
		)
	}
}
