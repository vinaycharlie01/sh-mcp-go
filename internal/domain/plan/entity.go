package plan

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/vinaycharlie01/sh-mcp-go/internal/domain/deployment"
)

// StepType classifies a plan step's action.
type StepType string

const (
	StepCreateNamespace   StepType = "CREATE_NAMESPACE"
	StepInstallCRDs       StepType = "INSTALL_CRDS"
	StepInstallDependency StepType = "INSTALL_DEPENDENCY"
	StepInstallChart      StepType = "INSTALL_CHART"
	StepUpgradeChart      StepType = "UPGRADE_CHART"
	StepRollbackChart     StepType = "ROLLBACK_CHART"
	StepUninstallChart    StepType = "UNINSTALL_CHART"
	StepInstallOperator   StepType = "INSTALL_OPERATOR"
	StepValidateReadiness StepType = "VALIDATE_READINESS"
	StepConfigureIngress  StepType = "CONFIGURE_INGRESS"
	StepConfigureStorage  StepType = "CONFIGURE_STORAGE"
	StepRunHealthCheck    StepType = "RUN_HEALTH_CHECK"
	StepRunSecurityScan   StepType = "RUN_SECURITY_SCAN"
)

// StepStatus tracks execution state of a single plan step.
type StepStatus string

const (
	StepStatusPending   StepStatus = "PENDING"
	StepStatusRunning   StepStatus = "RUNNING"
	StepStatusSucceeded StepStatus = "SUCCEEDED"
	StepStatusFailed    StepStatus = "FAILED"
	StepStatusSkipped   StepStatus = "SKIPPED"
)

// Step is a single atomic action within a deployment plan.
type Step struct {
	ID          string
	Type        StepType
	Description string
	Params      map[string]any
	Status      StepStatus
	DependsOn   []string
	Error       string
	StartedAt   *time.Time
	CompletedAt *time.Time
}

// PlanStatus is the overall execution state of a plan.
type PlanStatus string

const (
	PlanStatusDraft     PlanStatus = "DRAFT"
	PlanStatusApproved  PlanStatus = "APPROVED"
	PlanStatusExecuting PlanStatus = "EXECUTING"
	PlanStatusCompleted PlanStatus = "COMPLETED"
	PlanStatusFailed    PlanStatus = "FAILED"
	PlanStatusAborted   PlanStatus = "ABORTED"
)

// Plan is the aggregate root for an AI-generated deployment plan.
type Plan struct {
	id           string
	intent       string
	steps        []Step
	status       PlanStatus
	rollbackPlan *Plan
	createdAt    time.Time
	updatedAt    time.Time
	deploymentID *deployment.ID
}

// NewPlan constructs a deployment plan from a user intent string.
func NewPlan(intent string, steps []Step) *Plan {
	now := time.Now().UTC()
	return &Plan{
		id:        uuid.New().String(),
		intent:    intent,
		steps:     steps,
		status:    PlanStatusDraft,
		createdAt: now,
		updatedAt: now,
	}
}

// ID returns the plan identifier.
func (p *Plan) ID() string { return p.id }

// Intent returns the natural language intent that produced this plan.
func (p *Plan) Intent() string { return p.intent }

// Steps returns all steps in execution order.
func (p *Plan) Steps() []Step { return p.steps }

// Status returns current plan status.
func (p *Plan) Status() PlanStatus { return p.status }

// RollbackPlan returns the associated rollback plan, if set.
func (p *Plan) RollbackPlan() *Plan { return p.rollbackPlan }

// Approve marks the plan as approved for execution.
func (p *Plan) Approve() error {
	if p.status != PlanStatusDraft {
		return fmt.Errorf("plan can only be approved from DRAFT status, current: %s", p.status)
	}
	p.status = PlanStatusApproved
	p.updatedAt = time.Now().UTC()
	return nil
}

// Start transitions the plan to executing.
func (p *Plan) Start(deploymentID deployment.ID) error {
	if p.status != PlanStatusApproved {
		return fmt.Errorf("plan must be approved before starting, current: %s", p.status)
	}
	p.status = PlanStatusExecuting
	did := deploymentID
	p.deploymentID = &did
	p.updatedAt = time.Now().UTC()
	return nil
}

// UpdateStepStatus updates the execution status of a specific step.
func (p *Plan) UpdateStepStatus(stepID string, status StepStatus, errMsg string) error {
	for i := range p.steps {
		if p.steps[i].ID == stepID {
			p.steps[i].Status = status
			p.steps[i].Error = errMsg
			now := time.Now().UTC()
			switch status {
			case StepStatusPending:
				// no-op: pending steps have no timestamps
			case StepStatusRunning:
				p.steps[i].StartedAt = &now
			case StepStatusSucceeded, StepStatusFailed, StepStatusSkipped:
				p.steps[i].CompletedAt = &now
			}
			p.updatedAt = now
			return nil
		}
	}
	return fmt.Errorf("step %q not found in plan", stepID)
}

// Complete marks the entire plan as completed.
func (p *Plan) Complete() {
	p.status = PlanStatusCompleted
	p.updatedAt = time.Now().UTC()
}

// Fail marks the plan as failed.
func (p *Plan) Fail() {
	p.status = PlanStatusFailed
	p.updatedAt = time.Now().UTC()
}

// Abort marks the plan as aborted.
func (p *Plan) Abort() {
	p.status = PlanStatusAborted
	p.updatedAt = time.Now().UTC()
}

// SetRollbackPlan attaches a rollback plan to this plan.
func (p *Plan) SetRollbackPlan(rb *Plan) {
	p.rollbackPlan = rb
	p.updatedAt = time.Now().UTC()
}

// CreatedAt returns plan creation time.
func (p *Plan) CreatedAt() time.Time { return p.createdAt }

// UpdatedAt returns last update time.
func (p *Plan) UpdatedAt() time.Time { return p.updatedAt }

// DeploymentID returns the associated deployment ID after execution start.
func (p *Plan) DeploymentID() *deployment.ID { return p.deploymentID }

// ReadySteps returns all steps that are pending and whose dependencies are met.
func (p *Plan) ReadySteps() []Step {
	completed := make(map[string]bool)
	for _, s := range p.steps {
		if s.Status == StepStatusSucceeded || s.Status == StepStatusSkipped {
			completed[s.ID] = true
		}
	}

	var ready []Step
	for _, s := range p.steps {
		if s.Status != StepStatusPending {
			continue
		}
		allDepsmet := true
		for _, dep := range s.DependsOn {
			if !completed[dep] {
				allDepsmet = false

				break
			}
		}
		if allDepsmet {
			ready = append(ready, s)
		}
	}
	return ready
}

// NewStep creates a step with a generated ID.
func NewStep(stepType StepType, desc string, params map[string]any, dependsOn ...string) Step {
	return Step{
		ID:          uuid.New().String(),
		Type:        stepType,
		Description: desc,
		Params:      params,
		Status:      StepStatusPending,
		DependsOn:   dependsOn,
	}
}
