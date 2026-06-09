# Configuration Reference

The anonymizer is configured entirely through environment variables. This reference organizes all available variables by domain.

## Loading

Environment variables are parsed at startup using `caarlos0/env`. When using `task run`, the Taskfile automatically loads variables from `.env`. Copy `.env.example` to `.env` and adjust as needed.

## Server

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | HTTP server port |
| `HOST` | `0.0.0.0` | HTTP server host |
| `GRACEFUL_SHUTDOWN_TIMEOUT` | `30s` | Maximum time to wait for in-flight requests during shutdown |
| `MAX_BATCH_SIZE` | `100` | Maximum number of items per batch request |
| `LOG_LEVEL` | `INFO` | Logging level: `DEBUG`, `INFO`, `WARN`, or `ERROR` |
| `SERVICE_NAME` | `""` | Service name for OpenTelemetry |
| `PATTERN_MONITORING_ENABLED` | `false` | Enable leakspok pattern monitoring for debugging |

## Redis / Cache

Cache configuration uses the `PRIVACY_CACHE_` prefix. When caching is enabled, rule matching results are cached to avoid re-evaluating patterns on repeated input.

| Variable | Default | Description |
|----------|---------|-------------|
| `PRIVACY_CACHE_ENABLED` | `false` | Enable rule matching cache |
| `PRIVACY_CACHE_TTL` | `1h` | Cache entry time-to-live |
| `PRIVACY_CACHE_REDIS_ADDR` | `""` | Valkey/Redis address in `host:port` format |
| `PRIVACY_CACHE_REDIS_DISABLE_CLUSTER` | `false` | Use standalone client instead of cluster mode |
| `PRIVACY_CACHE_REDIS_DIAL_TIMEOUT` | `0` | Connection dial timeout |
| `PRIVACY_CACHE_REDIS_READ_TIMEOUT` | `0` | Socket read timeout |
| `PRIVACY_CACHE_REDIS_WRITE_TIMEOUT` | `0` | Socket write timeout |
| `PRIVACY_CACHE_REDIS_POOL_SIZE` | `0` | Maximum connections per CPU |
| `PRIVACY_CACHE_REDIS_MIN_IDLE_CONNS` | `0` | Minimum idle connections in the pool |
| `PRIVACY_CACHE_METRICS` | `true` | Expose cache-specific Prometheus metrics |
| `PRIVACY_CACHE_DISABLE_IN_MEMORY` | `false` | Disable in-memory server-assisted client-side caching |
| `REDIS_ANONYMIZER_SERVICE_V2_CACHE_TOKEN` | `""` | Redis AUTH password |

## Concurrency

Concurrency configuration uses the `PRIVACY_CONCURRENCY_` prefix. These options control parallel processing of tokens and rules within a single request.

| Variable | Default | Description |
|----------|---------|-------------|
| `PRIVACY_CONCURRENCY_ENABLED` | `false` | Enable concurrent processing |
| `PRIVACY_CONCURRENCY_TOKEN_PROCESSING` | `false` | Evaluate tokens in parallel |
| `PRIVACY_CONCURRENCY_RULE_PROCESSING` | `false` | Evaluate rules in parallel |
| `PRIVACY_CONCURRENCY_RULE_RUNNER_POOL_SIZE` | `0` | Goroutine pool size for rule runners |
| `PRIVACY_CONCURRENCY_TOKEN_POOL_SIZE` | `0` | Goroutine pool size for token processing |
| `PRIVACY_CONCURRENCY_MAX_GOROUTINE_IDLE_TIMEOUT` | `10s` | Time before idle goroutines are reclaimed |

Concurrency is most impactful for large text payloads where many tokens and rules need evaluation.

## OpenTelemetry

| Variable | Default | Description |
|----------|---------|-------------|
| `OTEL_ENABLED` | `false` | Enable OpenTelemetry tracing |
| `OTEL_EXPORTER_ADDR` | `localhost:4317` | OTel exporter address |

## See Also

- [Deployment Guide](./deployment.md) — production deployment and scaling
- [Getting Started](./getting-started.md) — quick start with defaults
- [Observability Guide](./observability.md) — metrics, tracing, and logging
