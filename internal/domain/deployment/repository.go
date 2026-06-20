package deployment

import "context"

// Repository is the persistence port for the Deployment aggregate.
// Implementations live in the adapters/storage layer.
type Repository interface {
	// Save persists or updates a Deployment. Idempotent on the same ID.
	Save(ctx context.Context, d *Deployment) error

	// FindByID retrieves a Deployment by its aggregate ID.
	FindByID(ctx context.Context, id ID) (*Deployment, error)

	// FindByReleaseName retrieves a Deployment by release name and namespace.
	FindByReleaseName(ctx context.Context, name ReleaseName, ns Namespace) (*Deployment, error)

	// ListByNamespace lists all deployments in a given namespace.
	ListByNamespace(ctx context.Context, ns Namespace) ([]*Deployment, error)

	// ListByStatus lists all deployments with the given status.
	ListByStatus(ctx context.Context, status Status) ([]*Deployment, error)

	// Delete removes a Deployment record permanently.
	Delete(ctx context.Context, id ID) error
}

// EventPublisher publishes domain events after a successful aggregate save.
type EventPublisher interface {
	Publish(ctx context.Context, events []Event) error
}
