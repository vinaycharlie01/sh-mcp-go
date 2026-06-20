package cluster

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/vinaycharlie01/sh-mcp-go/internal/domain/cluster"
	"github.com/vinaycharlie01/sh-mcp-go/internal/ports/outbound"
)

// Service provides cluster-level queries and validations.
type Service struct {
	k8s    outbound.KubernetesPort
	helm   outbound.HelmPort
	logger *slog.Logger
}

// NewService creates a new cluster service.
func NewService(k8s outbound.KubernetesPort, helm outbound.HelmPort, logger *slog.Logger) *Service {
	return &Service{k8s: k8s, helm: helm, logger: logger}
}

// GetInventory returns a full snapshot of the cluster state.
func (s *Service) GetInventory(ctx context.Context) (*cluster.ClusterInfo, error) {
	info, err := s.k8s.GetClusterInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting cluster info: %w", err)
	}

	releases, err := s.helm.ListReleases(ctx, "")
	if err != nil {
		s.logger.Warn("listing helm releases", slog.String("error", err.Error()))
	} else {
		for _, r := range releases {
			appVer := ""
			chartName := ""
			if r.Chart != nil && r.Chart.Metadata != nil {
				appVer = r.Chart.Metadata.AppVersion
				chartName = r.Chart.Metadata.Name + "-" + r.Chart.Metadata.Version
			}
			status := ""
			if r.Info != nil {
				status = string(r.Info.Status)
			}
			info.Releases = append(info.Releases, cluster.Release{
				Name:       r.Name,
				Namespace:  r.Namespace,
				Chart:      chartName,
				AppVersion: appVer,
				Status:     status,
				UpdatedAt:  r.Info.LastDeployed.Time,
			})
		}
	}

	return info, nil
}

// ValidateCluster runs cluster prerequisite checks.
func (s *Service) ValidateCluster(ctx context.Context) (*cluster.ValidationResult, error) {
	return s.k8s.ValidateCluster(ctx)
}

// GetHealthSummary returns health for resources in a namespace related to a release.
func (s *Service) GetHealthSummary(ctx context.Context, namespace, releaseName string) ([]cluster.ResourceHealth, error) {
	return s.k8s.GetResourceHealth(ctx, namespace, releaseName)
}

// EstimateResources returns resource estimates for a chart.
func (s *Service) EstimateResources(ctx context.Context, chartName, namespace string, replicas int) (*outbound.ResourceEstimate, error) {
	return s.k8s.EstimateResources(ctx, chartName, namespace, replicas)
}

// GenerateRCA generates a root cause analysis for a failing release.
func (s *Service) GenerateRCA(ctx context.Context, releaseName, namespace string) (string, error) {
	health, err := s.k8s.GetResourceHealth(ctx, namespace, releaseName)
	if err != nil {
		return "", fmt.Errorf("getting resource health: %w", err)
	}

	var analysis []string
	for _, h := range health {
		if h.Status != cluster.HealthStatusHealthy {
			analysis = append(analysis, fmt.Sprintf(
				"[%s] %s/%s: %s — %s",
				h.Status, h.Kind, h.Name, h.Message,
				rrca(h),
			))
		}
	}

	if len(analysis) == 0 {
		return fmt.Sprintf("Release %q in namespace %q appears healthy. No issues detected.", releaseName, namespace), nil
	}

	result := fmt.Sprintf("Root Cause Analysis for %q in %q:\n\n", releaseName, namespace)
	for i, a := range analysis {
		result += fmt.Sprintf("%d. %s\n", i+1, a)
	}
	return result, nil
}

// rrca returns a recommended remediation for a degraded resource.
func rrca(h cluster.ResourceHealth) string {
	switch h.Status {
	case cluster.HealthStatusDegraded:
		return "Check pod events with kubectl describe; consider increasing replica count or resource limits"
	case cluster.HealthStatusUnhealthy:
		return "Pod is crash-looping or OOMKilled; review logs and adjust memory limits"
	case cluster.HealthStatusHealthy:
		return "Resource is healthy; no action required"
	case cluster.HealthStatusUnknown:
		return "Status unknown; check cluster connectivity and node health"
	}

	return "Status unknown; check cluster connectivity and node health"
}
