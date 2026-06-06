package config

import "time"

// ServerConfig holds HTTP server configuration loaded from environment.
type ServerConfig struct {
	Port                    uint          `env:"PORT" envDefault:"8080"`
	Host                    string        `env:"HOST" envDefault:"0.0.0.0"`
	GracefulShutdownTimeout time.Duration `env:"GRACEFUL_SHUTDOWN_TIMEOUT" envDefault:"30s"`
}
