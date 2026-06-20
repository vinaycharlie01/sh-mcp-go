package deployment

import (
	"time"

	"github.com/google/uuid"
)

// EventType identifies a domain event.
type EventType string

const (
	EventTypeCreated         EventType = "deployment.created"
	EventTypeStarted         EventType = "deployment.started"
	EventTypeSucceeded       EventType = "deployment.succeeded"
	EventTypeFailed          EventType = "deployment.failed"
	EventTypeUpgradeStarted  EventType = "deployment.upgrade_started"
	EventTypeRollbackStarted EventType = "deployment.rollback_started"
	EventTypeRolledBack      EventType = "deployment.rolled_back"
)

// Event is the base domain event interface.
type Event interface {
	EventID() string
	AggregateID() ID
	Type() EventType
	OccurredAt() time.Time
}

type baseEvent struct {
	eventID     string
	aggregateID ID
	eventType   EventType
	occurredAt  time.Time
}

func (e baseEvent) EventID() string       { return e.eventID }
func (e baseEvent) AggregateID() ID       { return e.aggregateID }
func (e baseEvent) Type() EventType       { return e.eventType }
func (e baseEvent) OccurredAt() time.Time { return e.occurredAt }

func newBase(id ID, t EventType) baseEvent {
	return baseEvent{
		eventID:     uuid.New().String(),
		aggregateID: id,
		eventType:   t,
		occurredAt:  time.Now().UTC(),
	}
}

// DeploymentCreatedEvent is emitted when a Deployment is first created.
type DeploymentCreatedEvent struct {
	baseEvent
	ReleaseName ReleaseName
	Namespace   Namespace
	ChartRef    ChartReference
}

func NewDeploymentCreatedEvent(id ID, name ReleaseName, ns Namespace, chart ChartReference) DeploymentCreatedEvent {
	return DeploymentCreatedEvent{
		baseEvent:   newBase(id, EventTypeCreated),
		ReleaseName: name,
		Namespace:   ns,
		ChartRef:    chart,
	}
}

// DeploymentStartedEvent is emitted when deployment execution begins.
type DeploymentStartedEvent struct {
	baseEvent
	ReleaseName ReleaseName
	Namespace   Namespace
}

func NewDeploymentStartedEvent(id ID, name ReleaseName, ns Namespace) DeploymentStartedEvent {
	return DeploymentStartedEvent{
		baseEvent:   newBase(id, EventTypeStarted),
		ReleaseName: name,
		Namespace:   ns,
	}
}

// DeploymentSucceededEvent is emitted on successful deployment.
type DeploymentSucceededEvent struct {
	baseEvent
	ReleaseName  ReleaseName
	Namespace    Namespace
	HelmRevision int
}

func NewDeploymentSucceededEvent(id ID, name ReleaseName, ns Namespace, rev int) DeploymentSucceededEvent {
	return DeploymentSucceededEvent{
		baseEvent:    newBase(id, EventTypeSucceeded),
		ReleaseName:  name,
		Namespace:    ns,
		HelmRevision: rev,
	}
}

// DeploymentFailedEvent is emitted when deployment fails.
type DeploymentFailedEvent struct {
	baseEvent
	ReleaseName ReleaseName
	Namespace   Namespace
	Reason      string
}

func NewDeploymentFailedEvent(id ID, name ReleaseName, ns Namespace, reason string) DeploymentFailedEvent {
	return DeploymentFailedEvent{
		baseEvent:   newBase(id, EventTypeFailed),
		ReleaseName: name,
		Namespace:   ns,
		Reason:      reason,
	}
}

// DeploymentUpgradeStartedEvent is emitted when an upgrade begins.
type DeploymentUpgradeStartedEvent struct {
	baseEvent
	ReleaseName ReleaseName
	Namespace   Namespace
	NewChartRef ChartReference
}

func NewDeploymentUpgradeStartedEvent(id ID, name ReleaseName, ns Namespace, chart ChartReference) DeploymentUpgradeStartedEvent {
	return DeploymentUpgradeStartedEvent{
		baseEvent:   newBase(id, EventTypeUpgradeStarted),
		ReleaseName: name,
		Namespace:   ns,
		NewChartRef: chart,
	}
}

// DeploymentRollbackStartedEvent is emitted when rollback begins.
type DeploymentRollbackStartedEvent struct {
	baseEvent
	ReleaseName ReleaseName
	Namespace   Namespace
	ToVersion   int
}

func NewDeploymentRollbackStartedEvent(id ID, name ReleaseName, ns Namespace, toVersion int) DeploymentRollbackStartedEvent {
	return DeploymentRollbackStartedEvent{
		baseEvent:   newBase(id, EventTypeRollbackStarted),
		ReleaseName: name,
		Namespace:   ns,
		ToVersion:   toVersion,
	}
}

// DeploymentRolledBackEvent is emitted after a successful rollback.
type DeploymentRolledBackEvent struct {
	baseEvent
	ReleaseName ReleaseName
	Namespace   Namespace
	ToVersion   int
}

func NewDeploymentRolledBackEvent(id ID, name ReleaseName, ns Namespace, toVersion int) DeploymentRolledBackEvent {
	return DeploymentRolledBackEvent{
		baseEvent:   newBase(id, EventTypeRolledBack),
		ReleaseName: name,
		Namespace:   ns,
		ToVersion:   toVersion,
	}
}
