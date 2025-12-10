package daemon

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"go.uber.org/zap"

	"github.com/animus-coder/animus-coder/internal/agent"
	"github.com/animus-coder/animus-coder/internal/config"
	"github.com/animus-coder/animus-coder/internal/llm/configbuilder"
	"github.com/animus-coder/animus-coder/internal/observability"
	agentrpc "github.com/animus-coder/animus-coder/internal/rpc/agent"
	toolrpc "github.com/animus-coder/animus-coder/internal/rpc/tools"
	"github.com/animus-coder/animus-coder/internal/semantic"
	"github.com/animus-coder/animus-coder/internal/tools"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Server hosts lightweight daemon endpoints (health/metrics) and will later expose Agent RPC.
type Server struct {
	cfg     *config.Config
	logger  *zap.Logger
	runner  agentrpc.Runner
	metrics *observability.Metrics
	tools   *tools.Registry
}

// NewServer constructs a daemon instance.
func NewServer(cfg *config.Config, logger *zap.Logger) (*Server, error) {
	registry, err := configbuilder.BuildRegistryFromConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("build registry: %w", err)
	}

	agentCore := agent.New(registry, cfg.Agent)
	metrics := observability.NewMetrics()
	sandbox, err := tools.NewSandbox(cfg.Sandbox.WorkingDir, cfg.Sandbox, cfg.Tools)
	if err != nil {
		return nil, fmt.Errorf("build sandbox: %w", err)
	}
	gitTool := &tools.GitTool{
		WorkingDir: cfg.Sandbox.WorkingDir,
		AllowExec:  cfg.Tools.AllowGit && cfg.Sandbox.Enabled,
		DryRunOnly: !cfg.Sandbox.AllowWrite || !cfg.Tools.AllowFileWrite,
	}
	var semanticEngine *semantic.Engine
	if cfg.Tools.EnableSemantic {
		semanticEngine = semantic.NewEngine(sandbox.FS, cfg.Tools.SemanticMaxFiles, cfg.Tools.SemanticMaxFileBytes)
	}
	toolRegistry := tools.NewRegistry(sandbox.FS, sandbox.Terminal, gitTool, semanticEngine)
	strategy := agent.NewStrategyEngine(registry, cfg.Strategy)
	runner := &agentrpc.AgentRunner{Agent: agentCore, Metrics: metrics, Tools: toolRegistry, Strategy: strategy, Logger: logger}

	return &Server{cfg: cfg, logger: logger, runner: runner, metrics: metrics, tools: toolRegistry}, nil
}

// Run starts the HTTP server and blocks until context cancellation or fatal error.
func (s *Server) Run(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.healthHandler)
	mux.HandleFunc("/metrics", s.metricsHandler)
	mux.Handle("/tools/schemas", toolrpc.SchemaHandler{Registry: s.tools})

	switch strings.ToLower(strings.TrimSpace(s.cfg.Server.Transport)) {
	case "ndjson":
		mux.Handle("/agent/run", agentrpc.NewHandler(s.runner, s.metrics))
	default:
		path, handler := agentrpc.NewConnectHandler(s.runner, s.metrics)
		mux.Handle(path, handler)
		// keep legacy NDJSON path available during migration
		mux.Handle("/agent/run", agentrpc.NewHandler(s.runner, s.metrics))
	}

	handler := http.Handler(mux)
	if strings.ToLower(strings.TrimSpace(s.cfg.Server.Transport)) != "ndjson" {
		handler = h2c.NewHandler(handler, &http2.Server{})
	}

	server := &http.Server{
		Addr:              s.cfg.Server.Addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		s.logger.Info("starting mycodex daemon", zap.String("addr", s.cfg.Server.Addr))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		s.logger.Info("shutting down mycodex daemon")
	case err := <-errCh:
		return fmt.Errorf("server failed: %w", err)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("graceful shutdown failed: %w", err)
	}
	return nil
}

func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

func (s *Server) metricsHandler(w http.ResponseWriter, r *http.Request) {
	if !s.cfg.Server.MetricsEnabled {
		http.NotFound(w, r)
		return
	}

	promhttp.HandlerFor(s.metrics.Registry(), promhttp.HandlerOpts{}).ServeHTTP(w, r)
}
