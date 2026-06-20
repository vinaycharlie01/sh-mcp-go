package deployment

import (
	"time"

	"github.com/google/uuid"
)

// Status represents the lifecycle state of a deployment.
type Status string

const (
	StatusPending    Status = "PENDING"
	StatusPlanning   Status = "PLANNING"
	StatusValidating Status = "VALIDATING"
	StatusDeploying  Status = "DEPLOYING"
	StatusSucceeded  Status = "SUCCEEDED"
	StatusFailed     Status = "FAILED"
	StatusRollingBack Status = "ROLLING_BACK"
	StatusRolledBack Status = "ROLLED_BACK"
	StatusUpgrading  Status = "UPGRADING"
	StatusUninstalled Status = "UNINSTALLED"
)

// Deployment is the root aggregate for a Helm/Operator deployment lifecycle.
type Deployment struct {
	id          ID
	name        ReleaseName
	namespace   Namespace
	chartRef    ChartReference
	values      Values
	status      Status
	version     int
	history     []HistoryEntry
	events      []Event
	createdAt   time.Time
	updatedAt   time.Time
	annotations map[string]string
	labels      map[string]string
}

// New creates a new Deployment aggregate root.
func New(
	releaseName ReleaseName,
	ns Namespace,
	chart ChartReference,
	vals Values,
) (*Deployment, error) {
	if err := releaseName.Validate(); err != nil {
		return nil, err
	}
	if err := ns.Validate(); err != nil {
		return nil, err
	}
	if err := chart.Validate(); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	d := &Deployment{
		id:          ID(uuid.New().String()),
		name:        releaseName,
		namespace:   ns,
		chartRef:    chart,
		values:      vals,
		status:      StatusPending,
		version:     1,
		history:     make([]HistoryEntry, 0),
		events:      make([]Event, 0),
		createdAt:   now,
		updatedAt:   now,
		annotations: make(map[string]string),
		labels:      make(map[string]string),
	}
	d.record(NewDeploymentCreatedEvent(d.id, releaseName, ns, chart))

	return d, nil
}

// Reconstitute rebuilds a Deployment from persisted state (no event emission).
func Reconstitute(
	id ID,
	name ReleaseName,
	ns Namespace,
	chart ChartReference,
	vals Values,
	status Status,
	version int,
	history []HistoryEntry,
	createdAt, updatedAt time.Time,
	annotations, labels map[string]string,
) *Deployment {
	return &Deployment{
		id:          id,
		name:        name,
		namespace:   ns,
		chartRef:    chart,
		values:      vals,
		status:      status,
		version:     version,
		history:     history,
		events:      make([]Event, 0),
		createdAt:   createdAt,
		updatedAt:   updatedAt,
		annotations: annotations,
		labels:      labels,
	}
}

// --- Commands ---

// StartDeployment transitions the deployment into deploying state.
func (d *Deployment) StartDeployment() error {
	if d.status != StatusPending && d.status != StatusPlanning {
		return ErrInvalidTransition{From: d.status, To: StatusDeploying}
	}
	d.status = StatusDeploying
	d.updatedAt = time.Now().UTC()
	d.record(NewDeploymentStartedEvent(d.id, d.name, d.namespace))

	return nil
}

// MarkSucceeded transitions the deployment to succeeded.
func (d *Deployment) MarkSucceeded(helmRevision int) error {
	if d.status != StatusDeploying && d.status != StatusUpgrading {
		return ErrInvalidTransition{From: d.status, To: StatusSucceeded}
	}
	d.history = append(d.history, HistoryEntry{
		Version:     d.version,
		Status:      d.status,
		ChartRef:    d.chartRef,
		HelmVersion: helmRevision,
		Timestamp:   time.Now().UTC(),
	})
	d.status = StatusSucceeded
	d.version++
	d.updatedAt = time.Now().UTC()
	d.record(NewDeploymentSucceededEvent(d.id, d.name, d.namespace, helmRevision))

	return nil
}

// MarkFailed transitions the deployment to failed.
func (d *Deployment) MarkFailed(reason string) error {
	d.status = StatusFailed
	d.updatedAt = time.Now().UTC()
	d.record(NewDeploymentFailedEvent(d.id, d.name, d.namespace, reason))
	return nil
}

// StartUpgrade transitions to upgrading state with a new chart reference.
func (d *Deployment) StartUpgrade(newChart ChartReference, newValues Values) error {
	if d.status != StatusSucceeded {
		return ErrInvalidTransition{From: d.status, To: StatusUpgrading}
	}
	d.chartRef = newChart
	d.values = newValues
	d.status = StatusUpgrading
	d.updatedAt = time.Now().UTC()
	d.record(NewDeploymentUpgradeStartedEvent(d.id, d.name, d.namespace, newChart))
	return nil
}

// StartRollback transitions to rolling back state.
func (d *Deployment) StartRollback(toVersion int) error {
	if d.status != StatusFailed && d.status != StatusSucceeded {
		return ErrInvalidTransition{From: d.status, To: StatusRollingBack}
	}
	d.status = StatusRollingBack
	d.updatedAt = time.Now().UTC()
	d.record(NewDeploymentRollbackStartedEvent(d.id, d.name, d.namespace, toVersion))
	return nil
}

// MarkRolledBack completes the rollback.
func (d *Deployment) MarkRolledBack(toVersion int) {
	d.status = StatusRolledBack
	d.updatedAt = time.Now().UTC()
	d.record(NewDeploymentRolledBackEvent(d.id, d.name, d.namespace, toVersion))
}

// SetLabel adds or updates a label.
func (d *Deployment) SetLabel(key, val string) {
	d.labels[key] = val
	d.updatedAt = time.Now().UTC()
}

// SetAnnotation adds or updates an annotation.
func (d *Deployment) SetAnnotation(key, val string) {
	d.annotations[key] = val
	d.updatedAt = time.Now().UTC()
}

// --- Queries ---

func (d *Deployment) ID() ID                         { return d.id }
func (d *Deployment) Name() ReleaseName              { return d.name }
func (d *Deployment) Namespace() Namespace           { return d.namespace }
func (d *Deployment) ChartRef() ChartReference       { return d.chartRef }
func (d *Deployment) Values() Values                 { return d.values }
func (d *Deployment) Status() Status                 { return d.status }
func (d *Deployment) Version() int                   { return d.version }
func (d *Deployment) History() []HistoryEntry        { return d.history }
func (d *Deployment) CreatedAt() time.Time           { return d.createdAt }
func (d *Deployment) UpdatedAt() time.Time           { return d.updatedAt }
func (d *Deployment) Annotations() map[string]string { return d.annotations }
func (d *Deployment) Labels() map[string]string      { return d.labels }

func (d *Deployment) IsTerminal() bool {
	return d.status == StatusSucceeded ||
		d.status == StatusFailed ||
		d.status == StatusRolledBack ||
		d.status == StatusUninstalled
}

// DrainEvents returns and clears pending domain events.
func (d *Deployment) DrainEvents() []Event {
	evts := d.events
	d.events = make([]Event, 0)
	return evts
}

func (d *Deployment) record(e Event) {
	d.events = append(d.events, e)
}
