package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "modernc.org/sqlite" // register sqlite driver

	"github.com/vinaycharlie01/sh-mcp-go/internal/domain/deployment"
)

const schema = `
CREATE TABLE IF NOT EXISTS deployments (
    id              TEXT PRIMARY KEY,
    release_name    TEXT NOT NULL,
    namespace       TEXT NOT NULL,
    chart_name      TEXT NOT NULL,
    chart_repo      TEXT NOT NULL,
    chart_version   TEXT NOT NULL,
    chart_source    TEXT NOT NULL,
    values_json     TEXT NOT NULL DEFAULT '{}',
    status          TEXT NOT NULL,
    version         INTEGER NOT NULL DEFAULT 1,
    history_json    TEXT NOT NULL DEFAULT '[]',
    annotations_json TEXT NOT NULL DEFAULT '{}',
    labels_json     TEXT NOT NULL DEFAULT '{}',
    created_at      TEXT NOT NULL,
    updated_at      TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_deployments_release ON deployments(release_name, namespace);
CREATE INDEX IF NOT EXISTS idx_deployments_status ON deployments(status);
`

// Repository implements deployment.Repository using SQLite.
type Repository struct {
	db *sql.DB
}

// NewRepository opens a SQLite database and applies the schema.
func NewRepository(ctx context.Context, path string) (*Repository, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("opening sqlite: %w", err)
	}

	db.SetMaxOpenConns(1) // SQLite is single-writer
	db.SetMaxIdleConns(1)

	if _, err := db.ExecContext(ctx, schema); err != nil {
		return nil, fmt.Errorf("applying schema: %w", err)
	}

	return &Repository{db: db}, nil
}

// Close closes the underlying database connection.
func (r *Repository) Close() error { return r.db.Close() }

// Save inserts or replaces a Deployment record.
func (r *Repository) Save(ctx context.Context, d *deployment.Deployment) error {
	valuesJSON, err := json.Marshal(d.Values())
	if err != nil {
		return fmt.Errorf("marshaling values: %w", err)
	}
	historyJSON, err := json.Marshal(d.History())
	if err != nil {
		return fmt.Errorf("marshaling history: %w", err)
	}
	annotationsJSON, err := json.Marshal(d.Annotations())
	if err != nil {
		return fmt.Errorf("marshaling annotations: %w", err)
	}
	labelsJSON, err := json.Marshal(d.Labels())
	if err != nil {
		return fmt.Errorf("marshaling labels: %w", err)
	}

	chart := d.ChartRef()
	_, err = r.db.ExecContext(ctx, `
		INSERT INTO deployments
			(id, release_name, namespace, chart_name, chart_repo, chart_version, chart_source,
			 values_json, status, version, history_json, annotations_json, labels_json, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			status=excluded.status,
			version=excluded.version,
			chart_version=excluded.chart_version,
			values_json=excluded.values_json,
			history_json=excluded.history_json,
			annotations_json=excluded.annotations_json,
			labels_json=excluded.labels_json,
			updated_at=excluded.updated_at`,
		d.ID().String(),
		d.Name().String(),
		d.Namespace().String(),
		chart.Name,
		chart.RepoURL,
		chart.Version,
		string(chart.Source),
		string(valuesJSON),
		string(d.Status()),
		d.Version(),
		string(historyJSON),
		string(annotationsJSON),
		string(labelsJSON),
		d.CreatedAt().Format(time.RFC3339),
		d.UpdatedAt().Format(time.RFC3339),
	)

	return err
}

// FindByID retrieves a deployment by its aggregate ID.
func (r *Repository) FindByID(ctx context.Context, id deployment.ID) (*deployment.Deployment, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, release_name, namespace, chart_name, chart_repo, chart_version, chart_source,
		        values_json, status, version, history_json, annotations_json, labels_json, created_at, updated_at
		 FROM deployments WHERE id = ?`, id.String())

	return scanDeployment(row)
}

// FindByReleaseName retrieves a deployment by release name and namespace.
func (r *Repository) FindByReleaseName(
	ctx context.Context, name deployment.ReleaseName, ns deployment.Namespace,
) (*deployment.Deployment, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, release_name, namespace, chart_name, chart_repo, chart_version, chart_source,
		        values_json, status, version, history_json, annotations_json, labels_json, created_at, updated_at
		 FROM deployments WHERE release_name = ? AND namespace = ?`, name.String(), ns.String())

	return scanDeployment(row)
}

// ListByNamespace lists deployments in a namespace.
func (r *Repository) ListByNamespace(ctx context.Context, ns deployment.Namespace) ([]*deployment.Deployment, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, release_name, namespace, chart_name, chart_repo, chart_version, chart_source,
		        values_json, status, version, history_json, annotations_json, labels_json, created_at, updated_at
		 FROM deployments WHERE namespace = ?`, ns.String())
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	return scanDeployments(rows)
}

// ListByStatus lists deployments with the given status.
func (r *Repository) ListByStatus(ctx context.Context, status deployment.Status) ([]*deployment.Deployment, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, release_name, namespace, chart_name, chart_repo, chart_version, chart_source,
		        values_json, status, version, history_json, annotations_json, labels_json, created_at, updated_at
		 FROM deployments WHERE status = ?`, string(status))
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	return scanDeployments(rows)
}

// Delete removes a deployment record.
func (r *Repository) Delete(ctx context.Context, id deployment.ID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM deployments WHERE id = ?`, id.String())

	return err
}

type scanner interface {
	Scan(dest ...any) error
}

func scanDeployment(row scanner) (*deployment.Deployment, error) {
	var (
		id, releaseName, namespace                      string
		chartName, chartRepo, chartVersion, chartSource string
		valuesJSON, statusStr                           string
		version                                         int
		historyJSON, annotationsJSON, labelsJSON        string
		createdAtStr, updatedAtStr                      string
	)

	if err := row.Scan(
		&id, &releaseName, &namespace,
		&chartName, &chartRepo, &chartVersion, &chartSource,
		&valuesJSON, &statusStr, &version,
		&historyJSON, &annotationsJSON, &labelsJSON,
		&createdAtStr, &updatedAtStr,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("deployment not found: %w", err)
		}

		return nil, err
	}

	var values deployment.Values
	if err := json.Unmarshal([]byte(valuesJSON), &values); err != nil {
		values = make(deployment.Values)
	}

	var history []deployment.HistoryEntry
	_ = json.Unmarshal([]byte(historyJSON), &history)

	var annotations map[string]string
	if err := json.Unmarshal([]byte(annotationsJSON), &annotations); err != nil {
		annotations = make(map[string]string)
	}

	var labels map[string]string
	if err := json.Unmarshal([]byte(labelsJSON), &labels); err != nil {
		labels = make(map[string]string)
	}

	createdAt, _ := time.Parse(time.RFC3339, createdAtStr)
	updatedAt, _ := time.Parse(time.RFC3339, updatedAtStr)

	return deployment.Reconstitute(
		deployment.ID(id),
		deployment.ReleaseName(releaseName),
		deployment.Namespace(namespace),
		deployment.ChartReference{
			Name:    chartName,
			RepoURL: chartRepo,
			Version: chartVersion,
			Source:  deployment.ChartSource(chartSource),
		},
		values,
		deployment.Status(statusStr),
		version,
		history,
		createdAt, updatedAt,
		annotations, labels,
	), nil
}

func scanDeployments(rows *sql.Rows) ([]*deployment.Deployment, error) {
	var result []*deployment.Deployment
	for rows.Next() {
		d, err := scanDeployment(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, d)
	}

	return result, rows.Err()
}
