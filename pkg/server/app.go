package server

import (
	"github.com/Prosus-Cyber-Xchange/anonymizer/internal/handler"
	"github.com/Prosus-Cyber-Xchange/anonymizer/pkg/config"
	"github.com/Prosus-Cyber-Xchange/anonymizer/pkg/privacy"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/Prosus-Cyber-Xchange/leakspok/analyzer"
	leakspokmonitoring "github.com/Prosus-Cyber-Xchange/leakspok/monitoring"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/uber-go/tally/v4"
)

// AnonymizerServer is the configured anonymizer application.
// Created via NewFromConfig() and used to get an HTTP handler or start the built-in server.
type AnonymizerServer struct {
	logger           *slog.Logger
	byteAnalyzer     *analyzer.ByteAnalyzer
	plugins          []any
	envConfig        config.EnvConfig
	metricsScope     tally.Scope
	privacyService   *privacy.Service
	coreServices     CoreServices
	globalExceptions []privacy.ExceptionSettings
}

// NewFromConfig creates a new AnonymizerServer with the given context and options.
// It initializes all core services and wires any registered plugins.
func NewFromConfig(ctx context.Context, opts ...Option) (*AnonymizerServer, error) {
	envConfig, err := config.LoadEnv()
	if err != nil {
		return nil, fmt.Errorf("failed to load environment config: %w", err)
	}

	a := &AnonymizerServer{
		envConfig:    envConfig,
		metricsScope: tally.NoopScope,
	}

	// Apply options
	for _, opt := range opts {
		opt(a)
	}

	// Create default logger if not provided
	if a.logger == nil {
		a.logger = newLogger(envConfig)
	}

	// Create ByteAnalyzer if not provided, populating RunnerOptions from env config.
	if a.byteAnalyzer == nil {

		// Build RunnerOptions with Cache and Concurrency.
		runnerOpts := analyzer.RunnerOptions{
			Cache: analyzer.CacheOptions{
				Enabled:                 envConfig.Privacy.Cache,
				TTL:                     envConfig.Privacy.CacheTTL,
				DisableInMemoryCache:    envConfig.Privacy.DisableInMemoryCache,
				RedisAddr:               envConfig.Privacy.RedisCacheAddr,
				RedisPassword:           envConfig.RedisCacheToken,
				RedisDialTimeout:        envConfig.Privacy.RedisDialTimeout,
				RedisReadTimeout:        envConfig.Privacy.RedisReadTimeout,
				RedisWriteTimeout:       envConfig.Privacy.RedisWriteTimeout,
				RedisPoolSize:           envConfig.Privacy.RedisPoolSize,
				RedisMinIdleConns:       envConfig.Privacy.RedisMinIdleConns,
				RedisInsecureSkipVerify: true,
				RedisDisableClusterMode: envConfig.Privacy.RedisDisableCluster,
			},
		}

		runnerOpts.Concurrency = analyzer.ConcurrencyOptions{
			Enabled:                   envConfig.Privacy.ConcurrencyEnabled,
			ConcurrentTokenProcessing: envConfig.Privacy.ConcurrencyTokenProcessing,
			ConcurrentRuleProcessing:  envConfig.Privacy.ConcurrencyRuleProcessing,
			RuleRunnerPoolSize:        envConfig.Privacy.ConcurrencyRuleRunnerPoolSize,
			TokenPoolSize:             envConfig.Privacy.ConcurrencyTokenPoolSize,
			MaxGoroutineIdleTimeout:   envConfig.Privacy.ConcurrencyMaxGoroutineIdleTimeout,
		}

		ba, err := analyzer.MakeByteAnalyzer(ctx, a.logger, runnerOpts)
		if err != nil {
			return nil, fmt.Errorf("failed to create byte analyzer: %w", err)
		}
		a.byteAnalyzer = &ba
	}

	// Create PrivacyService (rules come from context or inline settings)
	a.privacyService = privacy.NewService(*a.byteAnalyzer, a.logger)

	return a, nil
}

func newLogger(envConfig config.EnvConfig) *slog.Logger {
	var level slog.Level
	switch strings.ToUpper(envConfig.LogLevel) {
	case "DEBUG":
		level = slog.LevelDebug
	case "WARN", "WARNING":
		level = slog.LevelWarn
	case "ERROR":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
}

// Handler returns the assembled HTTP handler.
// It includes the middleware stack, core CE endpoints, and any plugin routes.
// Call this to embed the anonymizer in a custom HTTP server.
func (a *AnonymizerServer) Handler() http.Handler {
	const healthPath = "/health"

	router := chi.NewRouter()

	router.Use(middleware.RealIP)
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			requestID := r.Header.Get(middleware.RequestIDHeader)
			if requestID == "" {
				requestID = uuid.NewString()
			}
			ctx = context.WithValue(ctx, "request_id", requestID)
			a.logger.DebugContext(ctx, "request", slog.String("request_id", requestID))
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})
	router.Use(middleware.Recoverer)
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := leakspokmonitoring.WithPatternMonitoring(r.Context(), a.envConfig.PatternMonitoringEnabled)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})

	router.Use(handler.AccessLog(a.logger, []string{healthPath}))

	// Plugin middleware (runs before core handler)
	coreServices := CoreServices{
		Logger:       a.logger,
		ByteAnalyzer: *a.byteAnalyzer,
	}
	for _, p := range a.plugins {
		if mr, ok := p.(MiddlewareRegistrar); ok {
			router.Use(mr.Middleware(coreServices))
		}
	}

	router.Get(healthPath, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
	})

	var h *handler.Handler
	if a.metricsScope != nil && a.metricsScope != tally.NoopScope {
		metrics := handler.NewPrivacyMetrics(a.metricsScope)
		h = handler.NewHandlerWithMetrics(a.logger, a.privacyService, a.envConfig.MaxBatchSize, a.globalExceptions, metrics)
	} else {
		h = handler.NewHandler(a.logger, a.privacyService, a.envConfig.MaxBatchSize, a.globalExceptions)
	}

	router.Route("/api/v1", func(r chi.Router) {
		r.Post("/anonymize", h.Anonymize)
		r.Post("/anonymize/batch", h.AnonymizeBatch)
	})

	return router
}

// ListenAndServe starts the HTTP server and blocks until ctx is cancelled.
// It handles graceful shutdown waiting up to Config.GracefulShutdownTimeout.
func (a *AnonymizerServer) ListenAndServe(ctx context.Context) error {
	addr := fmt.Sprintf("%s:%d", a.envConfig.Server.Host, a.envConfig.Server.Port)
	srv := &http.Server{
		Addr:    addr,
		Handler: a.Handler(),
	}

	errCh := make(chan error, 1)
	go func() {
		a.logger.InfoContext(ctx, "Starting HTTP server", slog.String("addr", addr))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), a.envConfig.Server.GracefulShutdownTimeout)
		defer cancel()
		a.logger.InfoContext(ctx, "Shutting down HTTP server gracefully")
		return srv.Shutdown(shutdownCtx)
	case err := <-errCh:
		return err
	}
}
