package server

import (
	"log/slog"
	"net/http"

	"github.com/Prosus-Cyber-Xchange/leakspok/analyzer"
)

// CoreServices provides shared dependencies to plugins.
type CoreServices struct {
	Logger       *slog.Logger
	ByteAnalyzer analyzer.ByteAnalyzer
}

// MiddlewareRegistrar is implemented by plugins that inject middleware
// into the HTTP request chain. The middleware runs before the core handler.
type MiddlewareRegistrar interface {
	Middleware(services CoreServices) func(http.Handler) http.Handler
}
