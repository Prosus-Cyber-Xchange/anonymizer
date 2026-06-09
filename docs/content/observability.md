# Observability Guide

## Logging

The anonymizer uses Go's standard library `log/slog` for structured logging.

- **Format:** JSON (production). Each log line is a JSON object with `time`, `level`, `msg`, and contextual attributes.
- **Levels:** `DEBUG`, `INFO`, `WARN`, `ERROR` — set via `LOG_LEVEL` env var (default: `INFO`).
- **What gets logged:**
  - Request start with `request_id`
  - Failed request body decoding
  - Invalid privacy settings
  - Anonymization errors
  - Batch size exceeded warnings
  - Server startup and shutdown events
  - Plugin events (plugin-specific)

**Example log output:**

```json
{"time":"2026-01-01T00:00:00Z","level":"INFO","msg":"Starting HTTP server","addr":"0.0.0.0:8080"}
```

## Metrics

Prometheus metrics are exposed at `/metrics` (scraped by Prometheus).

### Core Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `yggdrasil_authorizer_privacy_anonymization_duration` | Histogram | Anonymization operation duration in ms. Buckets: 1ms to 20s. |
| `yggdrasil_authorizer_privacy_anonymized_entity_count` | Counter | Count of anonymized entities, tagged by `entity` |
| `yggdrasil_authorizer_privacy_request_body_size` | Histogram | Request body size in bytes |

### Cache Metrics

When `PRIVACY_CACHE_METRICS=true` (default), cache operations are instrumented with Prometheus metrics covering hits, misses, and invalidation events.

### Access Logs

Every request is logged via chi middleware with method, path, status code, duration, and content type. The `/health` endpoint is excluded from access logging.

## Tracing

OpenTelemetry distributed tracing is available when `OTEL_ENABLED=true`.

- **Exporter:** Configured via `OTEL_EXPORTER_ADDR` (default: `localhost:4317`). Works with any OTLP-compatible collector (Jaeger, Grafana Tempo, Datadog Agent).
- **Tracer name:** `anonymizer-service`
- **Span names:**
  - `anonymize_json_request` — JSON anonymize request
  - `anonymize_textplain_request` — text/plain anonymize request
  - `anonymize_batch_request` — batch anonymize request
- **Span attributes:** entity count, content type, detected entities, error codes (on failure)

## Health Check

```
GET /health
```

Returns `200 OK` with `Content-Type: application/json`. This is a simple liveness probe — it does not check Redis connectivity.

```json
{}
```

## See Also

- [Configuration Reference](./configuration.md) — `LOG_LEVEL`, `OTEL_*`, and `PRIVACY_CACHE_METRICS`
- [Deployment Guide](./deployment.md) — readiness probes and health checks in K8s
