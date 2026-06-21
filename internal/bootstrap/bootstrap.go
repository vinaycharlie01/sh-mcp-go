package bootstrap

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/vinaycharlie01/sh-mcp-go/internal/adapters/events"
	helmadapter "github.com/vinaycharlie01/sh-mcp-go/internal/adapters/helm"
	k8sadapter "github.com/vinaycharlie01/sh-mcp-go/internal/adapters/kubernetes"
	mcpadapter "github.com/vinaycharlie01/sh-mcp-go/internal/adapters/mcp"
	"github.com/vinaycharlie01/sh-mcp-go/internal/adapters/observability"
	sqliterepo "github.com/vinaycharlie01/sh-mcp-go/internal/adapters/storage/sqlite"
	appcluster "github.com/vinaycharlie01/sh-mcp-go/internal/application/cluster"
	appdeployment "github.com/vinaycharlie01/sh-mcp-go/internal/application/deployment"
	appplanner "github.com/vinaycharlie01/sh-mcp-go/internal/application/planner"
	"github.com/vinaycharlie01/sh-mcp-go/internal/infrastructure/config"
	"github.com/vinaycharlie01/sh-mcp-go/internal/infrastructure/server"
	pkglogger "github.com/vinaycharlie01/sh-mcp-go/pkg/logger"
)

// App holds all wired application components.
type App struct {
	Config     *config.Config
	Logger     *slog.Logger
	MCPServer  *mcpadapter.Server
	HTTPServer *server.HTTPServer
	Metrics    *observability.Metrics
	Tracing    *observability.TracerProvider
	Storage    *sqliterepo.Repository
}

// Build assembles all application components using manual dependency injection.
// In a larger codebase, replace with Wire-generated code.
func Build(ctx context.Context, cfg *config.Config) (*App, error) {
	// Logger
	logger := pkglogger.New(cfg.Log.SlogLevel())
	logger.Info("bootstrapping sh-mcp-go")

	// Observability
	metrics, err := observability.NewMetrics(&cfg.Observability)
	if err != nil {
		return nil, fmt.Errorf("initializing metrics: %w", err)
	}

	tracing, err := observability.NewTracerProvider(ctx, &cfg.Observability)
	if err != nil {
		return nil, fmt.Errorf("initializing tracing: %w", err)
	}

	// Storage
	storage, err := sqliterepo.NewRepository(ctx, cfg.Storage.SQLite.Path)
	if err != nil {
		return nil, fmt.Errorf("initializing storage: %w", err)
	}
	logger.Info("storage initialized", slog.String("path", cfg.Storage.SQLite.Path))

	// Adapters
	helmClient, err := helmadapter.NewClient(&cfg.Helm, logger)
	if err != nil {
		return nil, fmt.Errorf("initializing helm client: %w", err)
	}

	k8sClient, err := k8sadapter.NewClient(&cfg.Kubernetes, logger)
	if err != nil {
		return nil, fmt.Errorf("initializing kubernetes client: %w", err)
	}

	eventPub := events.NewLogPublisher(logger)

	// Application services
	deploymentSvc := appdeployment.NewService(helmClient, k8sClient, storage, eventPub, logger)
	clusterSvc := appcluster.NewService(k8sClient, helmClient, logger)
	plannerSvc := appplanner.NewService(helmClient, k8sClient, logger)

	// MCP server
	mcpServer := mcpadapter.NewServer(&cfg.MCP, deploymentSvc, clusterSvc, plannerSvc, helmClient)

	// HTTP server
	httpServer := server.NewHTTPServer(&cfg.Server, metrics.Handler(), logger)

	return &App{
		Config:     cfg,
		Logger:     logger,
		MCPServer:  mcpServer,
		HTTPServer: httpServer,
		Metrics:    metrics,
		Tracing:    tracing,
		Storage:    storage,
	}, nil
}

// Shutdown gracefully tears down all components.
func (a *App) Shutdown(ctx context.Context) {
	a.Logger.Info("shutting down")

	if err := a.Tracing.Shutdown(ctx); err != nil {
		a.Logger.Error("tracing shutdown", slog.String("error", err.Error()))
	}
	if err := a.Metrics.Shutdown(ctx); err != nil {
		a.Logger.Error("metrics shutdown", slog.String("error", err.Error()))
	}
	if err := a.Storage.Close(); err != nil {
		a.Logger.Error("storage shutdown", slog.String("error", err.Error()))
	}
}
