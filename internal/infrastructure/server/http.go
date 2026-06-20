package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"

	pkglogger "github.com/vinaycharlie01/sh-mcp-go/pkg/logger"
	"github.com/vinaycharlie01/sh-mcp-go/pkg/version"
	"github.com/vinaycharlie01/sh-mcp-go/internal/infrastructure/config"
)

// HTTPServer wraps the standard http.Server with routing and middleware.
type HTTPServer struct {
	srv    *http.Server
	router *chi.Mux
	logger *slog.Logger
	cfg    *config.ServerConfig
}

// MetricsHandler is injected to serve /metrics.
type MetricsHandler interface {
	ServeHTTP(w http.ResponseWriter, r *http.Request)
}

// NewHTTPServer builds a fully configured HTTP server.
func NewHTTPServer(cfg *config.ServerConfig, metrics MetricsHandler, logger *slog.Logger) *HTTPServer {
	r := chi.NewRouter()

	r.Use(middleware.RealIP)
	r.Use(correlationIDMiddleware)
	r.Use(requestLoggerMiddleware(logger))
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(cfg.WriteTimeout))

	// Health endpoints
	r.Get("/healthz", healthz)
	r.Get("/readyz", readyz)
	r.Get("/livez", livez)
	r.Get("/version", versionHandler)

	// Metrics
	if metrics != nil {
		r.Handle("/metrics", metrics)
	}

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	return &HTTPServer{
		srv:    srv,
		router: r,
		logger: logger,
		cfg:    cfg,
	}
}

// Router returns the underlying chi router for adding additional routes.
func (s *HTTPServer) Router() *chi.Mux { return s.router }

// Start begins listening. Blocks until the context is cancelled.
func (s *HTTPServer) Start(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		s.logger.Info("HTTP server starting", slog.String("addr", s.srv.Addr))
		if err := s.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case <-ctx.Done():
		return s.shutdown(ctx)
	case err := <-errCh:
		return err
	}
}

// shutdown gracefully stops the server.
func (s *HTTPServer) shutdown(parentCtx context.Context) error {
	ctx, cancel := context.WithTimeout(parentCtx, s.cfg.ShutdownTimeout)
	defer cancel()
	s.logger.Info("HTTP server shutting down")

	return s.srv.Shutdown(ctx)
}

// --- Handlers ---

func healthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func readyz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ready"})
}

func livez(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "alive"})
}

func versionHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(version.Get())
}

// --- Middleware ---

func correlationIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cid := r.Header.Get("X-Correlation-ID")
		if cid == "" {
			cid = uuid.New().String()
		}
		rid := r.Header.Get("X-Request-ID")
		if rid == "" {
			rid = uuid.New().String()
		}
		ctx := pkglogger.WithCorrelationID(r.Context(), cid)
		ctx = pkglogger.WithRequestID(ctx, rid)
		w.Header().Set("X-Correlation-ID", cid)
		w.Header().Set("X-Request-ID", rid)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func requestLoggerMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ctx := r.Context()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			defer func() {
				log := pkglogger.FromContext(ctx, logger)
				log.Info("http request",
					slog.String("method", r.Method),
					slog.String("path", r.URL.Path),
					slog.Int("status", ww.Status()),
					slog.Duration("duration", time.Since(start)),
					slog.String("remote_addr", r.RemoteAddr),
				)
			}()

			next.ServeHTTP(ww, r)
		})
	}
}
