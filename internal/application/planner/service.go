package planner

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/vinaycharlie01/sh-mcp-go/internal/domain/plan"
	"github.com/vinaycharlie01/sh-mcp-go/internal/ports/outbound"
)

// DeploymentPlan is the output of the AI deployment planner.
type DeploymentPlan struct {
	PlanID      string
	Intent      string
	Steps       []plan.Step
	RollbackPlan *DeploymentPlan
	Warnings    []string
	EstimatedMins int
}

// Intent represents parsed deployment intent.
type Intent struct {
	Action    string // install, upgrade, rollback, uninstall
	Apps      []AppIntent
	Namespace string
}

// AppIntent is a single application within a broader intent.
type AppIntent struct {
	Name        string
	ChartName   string
	RepoURL     string
	Version     string
	Values      map[string]any
	Persistence bool
	Ingress     bool
	HA          bool
}

// Service is the AI deployment planner — it converts natural language intent
// into an ordered, dependency-resolved deployment plan.
type Service struct {
	helm   outbound.HelmPort
	k8s    outbound.KubernetesPort
	logger *slog.Logger
}

// NewService creates a new deployment planner service.
func NewService(helm outbound.HelmPort, k8s outbound.KubernetesPort, logger *slog.Logger) *Service {
	return &Service{helm: helm, k8s: k8s, logger: logger}
}

// Plan generates a deployment plan from the given natural language intent.
func (s *Service) Plan(ctx context.Context, intent string, namespace string) (*DeploymentPlan, error) {
	s.logger.Info("generating deployment plan", slog.String("intent", intent))

	parsed, err := s.parseIntent(intent, namespace)
	if err != nil {
		return nil, fmt.Errorf("parsing intent: %w", err)
	}

	steps, warnings, err := s.buildSteps(ctx, parsed)
	if err != nil {
		return nil, fmt.Errorf("building steps: %w", err)
	}

	rollback := s.buildRollbackPlan(parsed)
	domainPlan := plan.NewPlan(intent, steps)

	return &DeploymentPlan{
		PlanID:       domainPlan.ID(),
		Intent:       intent,
		Steps:        steps,
		RollbackPlan: rollback,
		Warnings:     warnings,
		EstimatedMins: estimateMinutes(steps),
	}, nil
}

// buildSteps converts parsed intent into ordered, dependency-aware steps.
func (s *Service) buildSteps(ctx context.Context, intent *Intent) ([]plan.Step, []string, error) {
	steps := make([]plan.Step, 0, len(intent.Apps)+2)
	var warnings []string

	// Validate cluster first
	result, err := s.k8s.ValidateCluster(ctx)
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("cluster validation warning: %v", err))
	} else if !result.Valid {
		return nil, nil, fmt.Errorf("cluster validation failed: %v", result.Errors)
	}

	// Step 1: Create namespace
	nsStep := plan.NewStep(plan.StepCreateNamespace,
		fmt.Sprintf("Create namespace %q", intent.Namespace),
		map[string]any{"namespace": intent.Namespace},
	)
	steps = append(steps, nsStep)

	// Per-app steps
	for _, app := range intent.Apps {
		// Resolve version
		version := app.Version
		if version == "" || version == "latest" {
			if resolved, resolveErr := s.helm.ResolveVersion(ctx, app.ChartName, app.RepoURL, ""); resolveErr == nil {
				version = resolved
			} else {
				warnings = append(warnings, fmt.Sprintf("could not resolve version for %s: %v", app.ChartName, resolveErr))
			}
		}

		// Validate chart
		validateStep := plan.NewStep(plan.StepInstallCRDs,
			fmt.Sprintf("Validate chart %s@%s", app.ChartName, version),
			map[string]any{
				"chart":   app.ChartName,
				"repo":    app.RepoURL,
				"version": version,
			},
			nsStep.ID,
		)
		steps = append(steps, validateStep)

		// Storage step if persistence requested
		var storageStepID string
		if app.Persistence {
			storageStep := plan.NewStep(plan.StepConfigureStorage,
				fmt.Sprintf("Configure persistent storage for %s", app.Name),
				map[string]any{
					"namespace": intent.Namespace,
					"app":       app.Name,
				},
				validateStep.ID,
			)
			steps = append(steps, storageStep)
			storageStepID = storageStep.ID
		}

		// Install chart
		dependsOn := []string{validateStep.ID}
		if storageStepID != "" {
			dependsOn = append(dependsOn, storageStepID)
		}

		actionType := plan.StepInstallChart
		if intent.Action == "upgrade" {
			actionType = plan.StepUpgradeChart
		}

		values := app.Values
		if values == nil {
			values = make(map[string]any)
		}
		if app.HA {
			values["replicaCount"] = haReplicaCount
		}
		if app.Persistence {
			values["persistence.enabled"] = true
		}

		installStep := plan.NewStep(actionType,
			fmt.Sprintf("%s %s@%s in %s", cases.Title(language.English).String(intent.Action), app.ChartName, version, intent.Namespace),
			map[string]any{
				"release_name": app.Name,
				"namespace":    intent.Namespace,
				"chart":        app.ChartName,
				"repo":         app.RepoURL,
				"version":      version,
				"values":       values,
			},
			dependsOn...,
		)
		steps = append(steps, installStep)

		// Ingress step
		if app.Ingress {
			ingressStep := plan.NewStep(plan.StepConfigureIngress,
				fmt.Sprintf("Configure ingress for %s", app.Name),
				map[string]any{
					"namespace": intent.Namespace,
					"app":       app.Name,
				},
				installStep.ID,
			)
			steps = append(steps, ingressStep)
		}

		// Health check
		healthStep := plan.NewStep(plan.StepValidateReadiness,
			fmt.Sprintf("Validate readiness of %s", app.Name),
			map[string]any{
				"release_name": app.Name,
				"namespace":    intent.Namespace,
			},
			installStep.ID,
		)
		steps = append(steps, healthStep)
	}

	return steps, warnings, nil
}

// buildRollbackPlan generates a rollback plan for the deployment.
func (s *Service) buildRollbackPlan(intent *Intent) *DeploymentPlan {
	rollbackSteps := make([]plan.Step, 0, len(intent.Apps))
	for _, app := range intent.Apps {
		rollbackSteps = append(rollbackSteps, plan.NewStep(
			plan.StepRollbackChart,
			fmt.Sprintf("Rollback %s to previous version", app.Name),
			map[string]any{
				"release_name": app.Name,
				"namespace":    intent.Namespace,
				"version":      0,
			},
		))
	}
	return &DeploymentPlan{
		Intent: "rollback: " + intent.Namespace,
		Steps:  rollbackSteps,
	}
}

// parseIntent converts natural language into structured Intent.
// In production this would use an LLM; here we use rule-based parsing.
func (s *Service) parseIntent(intent, namespace string) (*Intent, error) {
	lower := strings.ToLower(intent)

	action := "install"
	if strings.Contains(lower, "upgrade") || strings.Contains(lower, "update") {
		action = "upgrade"
	} else if strings.Contains(lower, "rollback") || strings.Contains(lower, "revert") {
		action = "rollback"
	} else if strings.Contains(lower, "uninstall") || strings.Contains(lower, "remove") || strings.Contains(lower, "delete") {
		action = "uninstall"
	}

	ns := namespace
	if ns == "" {
		ns = "default"
	}

	apps := detectApps(lower)
	if len(apps) == 0 {
		return nil, fmt.Errorf("no known applications detected in intent: %q", intent)
	}

	return &Intent{
		Action:    action,
		Apps:      apps,
		Namespace: ns,
	}, nil
}

// detectApps maps well-known application names from natural language.
func detectApps(intent string) []AppIntent {
	catalogue := []struct {
		keywords    []string
		app         AppIntent
	}{
		{
			keywords: []string{"prometheus"},
			app: AppIntent{
				Name:      "prometheus",
				ChartName: "kube-prometheus-stack",
				RepoURL:   "https://prometheus-community.github.io/helm-charts",
				Persistence: true,
			},
		},
		{
			keywords: []string{"grafana"},
			app: AppIntent{
				Name:      "grafana",
				ChartName: "grafana",
				RepoURL:   "https://grafana.github.io/helm-charts",
				Persistence: true,
				Ingress:    true,
			},
		},
		{
			keywords: []string{"redis"},
			app: AppIntent{
				Name:      "redis",
				ChartName: "redis",
				RepoURL:   "https://charts.bitnami.com/bitnami",
				Persistence: true,
				HA:         strings.Contains(intent, "ha") || strings.Contains(intent, "high availability"),
			},
		},
		{
			keywords: []string{"cert-manager", "cert manager", "certmanager"},
			app: AppIntent{
				Name:      "cert-manager",
				ChartName: "cert-manager",
				RepoURL:   "https://charts.jetstack.io",
			},
		},
		{
			keywords: []string{"nginx", "ingress"},
			app: AppIntent{
				Name:      "ingress-nginx",
				ChartName: "ingress-nginx",
				RepoURL:   "https://kubernetes.github.io/ingress-nginx",
			},
		},
		{
			keywords: []string{"istio"},
			app: AppIntent{
				Name:      "istiod",
				ChartName: "istiod",
				RepoURL:   "https://istio-release.storage.googleapis.com/charts",
			},
		},
		{
			keywords: []string{"loki"},
			app: AppIntent{
				Name:      "loki",
				ChartName: "loki",
				RepoURL:   "https://grafana.github.io/helm-charts",
				Persistence: true,
			},
		},
		{
			keywords: []string{"velero"},
			app: AppIntent{
				Name:      "velero",
				ChartName: "velero",
				RepoURL:   "https://vmware-tanzu.github.io/helm-charts",
			},
		},
		{
			keywords: []string{"argocd", "argo cd"},
			app: AppIntent{
				Name:      "argocd",
				ChartName: "argo-cd",
				RepoURL:   "https://argoproj.github.io/argo-helm",
				Ingress:    true,
			},
		},
	}

	var found []AppIntent
	for _, entry := range catalogue {
		for _, kw := range entry.keywords {
			if strings.Contains(intent, kw) {
				found = append(found, entry.app)

				break
			}
		}
	}
	return found
}

const (
	minutesPerStep  = 2
	haReplicaCount  = 3
)

func estimateMinutes(steps []plan.Step) int {
	return len(steps) * minutesPerStep
}
