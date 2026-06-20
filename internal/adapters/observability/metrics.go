package observability

import (
	"context"
	"net/http"
	"net/http/pprof"

	"github.com/go-chi/chi/v5"
	promclient "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	otelprom "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"

	"github.com/vinaycharlie01/sh-mcp-go/internal/infrastructure/config"
)

// Metrics holds all instrumentation for sh-mcp-go.
type Metrics struct {
	provider *sdkmetric.MeterProvider
	registry *promclient.Registry

	DeploymentTotal       metric.Int64Counter
	DeploymentFailures    metric.Int64Counter
	HelmOperationTotal    metric.Int64Counter
	DeploymentDuration    metric.Float64Histogram
	HelmOperationDuration metric.Float64Histogram
	ActiveDeployments     metric.Int64UpDownCounter
}

// NewMetrics initialises the OpenTelemetry metric provider with Prometheus exporter.
func NewMetrics(cfg *config.ObservabilityConfig) (*Metrics, error) {
	registry := promclient.NewRegistry()
	registry.MustRegister(collectors.NewGoCollector())
	registry.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))

	exporter, err := otelprom.New(otelprom.WithRegisterer(registry))
	if err != nil {
		return nil, err
	}

	provider := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(exporter),
	)
	otel.SetMeterProvider(provider)

	meter := provider.Meter(cfg.ServiceName)

	depTotal, _ := meter.Int64Counter("shmcp_deployments_total",
		metric.WithDescription("Total deployment operations"))
	depFailures, _ := meter.Int64Counter("shmcp_deployment_failures_total",
		metric.WithDescription("Total failed deployments"))
	helmOps, _ := meter.Int64Counter("shmcp_helm_operations_total",
		metric.WithDescription("Total Helm SDK operations"))
	depDur, _ := meter.Float64Histogram("shmcp_deployment_duration_seconds",
		metric.WithDescription("Deployment duration in seconds"))
	helmDur, _ := meter.Float64Histogram("shmcp_helm_operation_duration_seconds",
		metric.WithDescription("Helm operation duration in seconds"))
	activeDep, _ := meter.Int64UpDownCounter("shmcp_active_deployments",
		metric.WithDescription("In-progress deployments"))

	return &Metrics{
		provider:              provider,
		registry:              registry,
		DeploymentTotal:       depTotal,
		DeploymentFailures:    depFailures,
		HelmOperationTotal:    helmOps,
		DeploymentDuration:    depDur,
		HelmOperationDuration: helmDur,
		ActiveDeployments:     activeDep,
	}, nil
}

// Shutdown flushes and shuts down the metrics provider.
func (m *Metrics) Shutdown(ctx context.Context) error {
	return m.provider.Shutdown(ctx)
}

// Handler returns an http.Handler that serves Prometheus metrics.
func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{
		EnableOpenMetrics: true,
	})
}

// RegisterProfilingRoutes adds pprof endpoints to a chi router.
func RegisterProfilingRoutes(r chi.Router) {
	r.HandleFunc("/debug/pprof/", http.HandlerFunc(pprof.Index))
	r.HandleFunc("/debug/pprof/cmdline", http.HandlerFunc(pprof.Cmdline))
	r.HandleFunc("/debug/pprof/profile", http.HandlerFunc(pprof.Profile))
	r.HandleFunc("/debug/pprof/symbol", http.HandlerFunc(pprof.Symbol))
	r.HandleFunc("/debug/pprof/trace", http.HandlerFunc(pprof.Trace))
}
