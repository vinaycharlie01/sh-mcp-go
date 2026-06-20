package deployment

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/vinaycharlie01/sh-mcp-go/internal/domain/deployment"
	"github.com/vinaycharlie01/sh-mcp-go/internal/ports/outbound"
)

// Service orchestrates chart install/upgrade/rollback/uninstall workflows.
// It coordinates the Helm port, Kubernetes port, storage, and domain aggregates.
type Service struct {
	helm   outbound.HelmPort
	k8s    outbound.KubernetesPort
	store  outbound.DeploymentStore
	pub    deployment.EventPublisher
	logger *slog.Logger
}

// NewService constructs the deployment application service.
func NewService(
	helm outbound.HelmPort,
	k8s outbound.KubernetesPort,
	store outbound.DeploymentStore,
	pub deployment.EventPublisher,
	logger *slog.Logger,
) *Service {
	return &Service{
		helm:   helm,
		k8s:    k8s,
		store:  store,
		pub:    pub,
		logger: logger,
	}
}

// InstallChart installs a Helm chart onto the cluster.
func (s *Service) InstallChart(ctx context.Context, cmd InstallChartCommand) (*InstallChartResult, error) {
	log := s.logger.With(
		slog.String("release", cmd.ReleaseName),
		slog.String("namespace", cmd.Namespace),
		slog.String("chart", cmd.ChartName),
	)
	log.Info("install chart command received")

	// Build domain aggregate
	chartRef := deployment.ChartReference{
		Name:    cmd.ChartName,
		RepoURL: cmd.RepoURL,
		Version: cmd.Version,
		Source:  deployment.ChartSourceRepo,
	}

	agg, err := deployment.New(
		deployment.ReleaseName(cmd.ReleaseName),
		deployment.Namespace(cmd.Namespace),
		chartRef,
		deployment.Values(cmd.Values),
	)
	if err != nil {
		return nil, fmt.Errorf("creating deployment aggregate: %w", err)
	}

	// Ensure namespace exists
	if cmd.CreateNS || !cmd.DryRun {
		if nsErr := s.k8s.EnsureNamespace(ctx, outbound.NamespaceSpec{Name: cmd.Namespace}); nsErr != nil {
			return nil, fmt.Errorf("ensuring namespace: %w", nsErr)
		}
	}

	// Save initial state
	if saveErr := s.store.Save(ctx, agg); saveErr != nil {
		return nil, fmt.Errorf("saving deployment: %w", saveErr)
	}

	// Transition to deploying
	if err := agg.StartDeployment(); err != nil {
		return nil, err
	}
	if saveErr := s.store.Save(ctx, agg); saveErr != nil {
		return nil, saveErr
	}

	// Execute Helm install
	rel, err := s.helm.Install(ctx, outbound.HelmInstallRequest{
		ReleaseName: cmd.ReleaseName,
		Namespace:   cmd.Namespace,
		ChartName:   cmd.ChartName,
		RepoURL:     cmd.RepoURL,
		Version:     cmd.Version,
		Values:      cmd.Values,
		DryRun:      cmd.DryRun,
		Wait:        cmd.Wait,
		Atomic:      cmd.Atomic,
		CreateNS:    cmd.CreateNS,
		Timeout: cmd.TimeoutSecs,
	})
	if err != nil {
		_ = agg.MarkFailed(err.Error())
		_ = s.store.Save(ctx, agg)
		s.publishEvents(ctx, agg)
		return nil, fmt.Errorf("helm install: %w", err)
	}

	// Mark succeeded
	if markErr := agg.MarkSucceeded(rel.Version); markErr != nil {
		return nil, markErr
	}
	if saveErr := s.store.Save(ctx, agg); saveErr != nil {
		return nil, saveErr
	}

	s.publishEvents(ctx, agg)

	notes := ""
	if rel.Info != nil {
		notes = rel.Info.Notes
	}

	log.Info("chart installed successfully", slog.Int("revision", rel.Version))
	return &InstallChartResult{
		DeploymentID: agg.ID().String(),
		ReleaseName:  rel.Name,
		Namespace:    rel.Namespace,
		Revision:     rel.Version,
		Status:       agg.Status(),
		Notes:        notes,
	}, nil
}

// UpgradeChart upgrades an existing Helm release.
func (s *Service) UpgradeChart(ctx context.Context, cmd UpgradeChartCommand) (*UpgradeChartResult, error) {
	log := s.logger.With(
		slog.String("release", cmd.ReleaseName),
		slog.String("namespace", cmd.Namespace),
	)
	log.Info("upgrade chart command received")

	// Find existing deployment
	agg, err := s.store.FindByReleaseName(ctx,
		deployment.ReleaseName(cmd.ReleaseName),
		deployment.Namespace(cmd.Namespace),
	)
	if err != nil {
		return nil, fmt.Errorf("finding deployment: %w", err)
	}

	newChart := deployment.ChartReference{
		Name:    cmd.ChartName,
		RepoURL: cmd.RepoURL,
		Version: cmd.Version,
		Source:  deployment.ChartSourceRepo,
	}
	if err := agg.StartUpgrade(newChart, deployment.Values(cmd.Values)); err != nil {
		return nil, err
	}
	if err := s.store.Save(ctx, agg); err != nil {
		return nil, err
	}

	rel, err := s.helm.Upgrade(ctx, outbound.HelmUpgradeRequest{
		ReleaseName: cmd.ReleaseName,
		Namespace:   cmd.Namespace,
		ChartName:   cmd.ChartName,
		RepoURL:     cmd.RepoURL,
		Version:     cmd.Version,
		Values:      cmd.Values,
		DryRun:      cmd.DryRun,
		Wait:        cmd.Wait,
		Atomic:      cmd.Atomic,
		ReuseValues: cmd.ReuseValues,
		ResetValues: cmd.ResetValues,
		Force:       cmd.Force,
		Timeout: cmd.TimeoutSecs,
	})
	if err != nil {
		_ = agg.MarkFailed(err.Error())
		_ = s.store.Save(ctx, agg)
		s.publishEvents(ctx, agg)
		return nil, fmt.Errorf("helm upgrade: %w", err)
	}

	if err := agg.MarkSucceeded(rel.Version); err != nil {
		return nil, err
	}
	if err := s.store.Save(ctx, agg); err != nil {
		return nil, err
	}
	s.publishEvents(ctx, agg)

	notes := ""
	if rel.Info != nil {
		notes = rel.Info.Notes
	}

	log.Info("chart upgraded", slog.Int("revision", rel.Version))
	return &UpgradeChartResult{
		DeploymentID: agg.ID().String(),
		ReleaseName:  rel.Name,
		Namespace:    rel.Namespace,
		Revision:     rel.Version,
		Status:       agg.Status(),
		Notes:        notes,
	}, nil
}

// RollbackChart rolls back a Helm release.
func (s *Service) RollbackChart(ctx context.Context, cmd RollbackChartCommand) error {
	s.logger.Info("rollback chart",
		slog.String("release", cmd.ReleaseName),
		slog.Int("version", cmd.Version),
	)

	agg, err := s.store.FindByReleaseName(ctx,
		deployment.ReleaseName(cmd.ReleaseName),
		deployment.Namespace(cmd.Namespace),
	)
	if err != nil {
		return fmt.Errorf("finding deployment: %w", err)
	}

	if err := agg.StartRollback(cmd.Version); err != nil {
		return err
	}
	if err := s.store.Save(ctx, agg); err != nil {
		return err
	}

	if err := s.helm.Rollback(ctx, outbound.HelmRollbackRequest{
		ReleaseName: cmd.ReleaseName,
		Namespace:   cmd.Namespace,
		Version:     cmd.Version,
		DryRun:      cmd.DryRun,
		Wait:        cmd.Wait,
		Timeout: cmd.TimeoutSecs,
	}); err != nil {
		_ = agg.MarkFailed(err.Error())
		_ = s.store.Save(ctx, agg)
		s.publishEvents(ctx, agg)
		return fmt.Errorf("helm rollback: %w", err)
	}

	agg.MarkRolledBack(cmd.Version)
	if err := s.store.Save(ctx, agg); err != nil {
		return err
	}
	s.publishEvents(ctx, agg)
	return nil
}

// UninstallChart removes a Helm release.
func (s *Service) UninstallChart(ctx context.Context, cmd UninstallChartCommand) error {
	s.logger.Info("uninstall chart", slog.String("release", cmd.ReleaseName))

	if err := s.helm.Uninstall(ctx, outbound.HelmUninstallRequest{
		ReleaseName: cmd.ReleaseName,
		Namespace:   cmd.Namespace,
		DryRun:      cmd.DryRun,
		KeepHistory: cmd.KeepHistory,
		Timeout: cmd.TimeoutSecs,
	}); err != nil {
		return fmt.Errorf("helm uninstall: %w", err)
	}

	agg, err := s.store.FindByReleaseName(ctx,
		deployment.ReleaseName(cmd.ReleaseName),
		deployment.Namespace(cmd.Namespace),
	)
	if err == nil {
		_ = s.store.Delete(ctx, agg.ID())
	}
	return nil
}

func (s *Service) publishEvents(ctx context.Context, agg *deployment.Deployment) {
	events := agg.DrainEvents()
	if s.pub != nil && len(events) > 0 {
		if err := s.pub.Publish(ctx, events); err != nil {
			s.logger.Warn("publishing domain events failed", slog.String("error", err.Error()))
		}
	}
}
