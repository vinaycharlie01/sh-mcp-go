package outbound

import (
	"context"

	"github.com/vinaycharlie01/sh-mcp-go/internal/domain/deployment"
	"github.com/vinaycharlie01/sh-mcp-go/internal/domain/plan"
)

// DeploymentStore is the outbound persistence port for Deployment aggregates.
// This is an alias for the repository defined in the domain.
type DeploymentStore = deployment.Repository

// PlanStore is the outbound persistence port for Plan aggregates.
type PlanStore interface {
	// SavePlan persists or updates a deployment plan.
	SavePlan(ctx context.Context, p *plan.Plan) error

	// FindPlan retrieves a plan by its ID.
	FindPlan(ctx context.Context, id string) (*plan.Plan, error)

	// ListPlans lists the most recent N plans, newest first.
	ListPlans(ctx context.Context, limit int) ([]*plan.Plan, error)

	// DeletePlan removes a plan by ID.
	DeletePlan(ctx context.Context, id string) error
}

// Transactor provides unit-of-work transaction support.
type Transactor interface {
	// WithTransaction executes fn within a database transaction.
	// If fn returns an error, the transaction is rolled back.
	WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}
