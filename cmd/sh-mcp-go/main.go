package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/sync/errgroup"

	"github.com/vinaycharlie01/sh-mcp-go/internal/bootstrap"
	"github.com/vinaycharlie01/sh-mcp-go/internal/infrastructure/config"
	pkglogger "github.com/vinaycharlie01/sh-mcp-go/pkg/logger"
	"github.com/vinaycharlie01/sh-mcp-go/pkg/version"
)

func main() {
	if err := run(); err != nil {

		
		fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Bootstrap logger early for startup errors.
	startupLogger := pkglogger.New(slog.LevelInfo)
	startupLogger.Info("starting sh-mcp-go", slog.String("version", version.Get().String()))

	// Load configuration.
	loader := config.NewLoader()
	cfg, err := loader.Load()
	if err != nil {
		return fmt.Errorf("loading configuration: %w", err)
	}

	// Root context tied to OS signals.
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Wire all components.
	app, err := bootstrap.Build(ctx, cfg)
	if err != nil {
		return fmt.Errorf("building application: %w", err)
	}
	defer app.Shutdown(context.Background())

	slog.Info("application ready",
		slog.String("mcp_transport", cfg.MCP.Transport),
		slog.String("server_addr", fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)),
	)

	// Run components concurrently; stop all on first error or signal.
	g, gctx := errgroup.WithContext(ctx)

	// MCP server (primary)
	g.Go(func() error {
		switch cfg.MCP.Transport {
		case "stdio":
			return app.MCPServer.ServeStdio(gctx)
		case "sse":
			return app.MCPServer.ServeSSE(gctx)
		default:
			return fmt.Errorf("unknown MCP transport: %s", cfg.MCP.Transport)
		}
	})

	// HTTP server (health, metrics, admin)
	g.Go(func() error {
		return app.HTTPServer.Start(gctx)
	})

	if err := g.Wait(); err != nil && err != context.Canceled {
		return fmt.Errorf("runtime error: %w", err)
	}

	slog.Info("sh-mcp-go stopped cleanly")

	return nil
}
