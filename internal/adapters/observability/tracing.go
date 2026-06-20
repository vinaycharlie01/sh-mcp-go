package observability

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/vinaycharlie01/sh-mcp-go/internal/infrastructure/config"
)

// TracerProvider wraps the OTel SDK tracer provider.
type TracerProvider struct {
	provider *sdktrace.TracerProvider
}

// NewTracerProvider creates an OTel tracer provider with OTLP/HTTP export.
func NewTracerProvider(ctx context.Context, cfg *config.ObservabilityConfig) (*TracerProvider, error) {
	if !cfg.TracingEnabled {
		// Return a no-op provider when tracing is disabled.
		return &TracerProvider{provider: sdktrace.NewTracerProvider()}, nil
	}

	exporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(cfg.OTLPEndpoint),
		otlptracehttp.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("creating OTLP exporter: %w", err)
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("creating OTel resource: %w", err)
	}

	sampler := sdktrace.TraceIDRatioBased(cfg.SamplingRate)
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
	)

	otel.SetTracerProvider(provider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return &TracerProvider{provider: provider}, nil
}

// Tracer returns a named tracer from the provider.
func (t *TracerProvider) Tracer(name string) trace.Tracer {
	return t.provider.Tracer(name)
}

// Shutdown flushes pending spans.
func (t *TracerProvider) Shutdown(ctx context.Context) error {
	return t.provider.Shutdown(ctx)
}
