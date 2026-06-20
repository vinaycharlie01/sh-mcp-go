package events

import (
	"context"
	"log/slog"

	"github.com/vinaycharlie01/sh-mcp-go/internal/domain/deployment"
)

// LogPublisher is a deployment.EventPublisher that logs domain events.
// In production, replace or augment with a message broker (Kafka, NATS, etc.).
type LogPublisher struct {
	logger *slog.Logger
}

// NewLogPublisher creates a publisher that writes events to structured logs.
func NewLogPublisher(logger *slog.Logger) *LogPublisher {
	return &LogPublisher{logger: logger}
}

// Publish logs all domain events at DEBUG level.
func (p *LogPublisher) Publish(_ context.Context, events []deployment.Event) error {
	for _, e := range events {
		p.logger.Debug("domain event",
			slog.String("event_id", e.EventID()),
			slog.String("event_type", string(e.Type())),
			slog.String("aggregate_id", e.AggregateID().String()),
			slog.Time("occurred_at", e.OccurredAt()),
		)
	}
	return nil
}
