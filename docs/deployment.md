# Deployment Guide

## Docker

### Build

```bash
docker build -t anonymizer .
```

The provided `Dockerfile` uses a multi-stage build: it compiles a static binary in a `golang:1.25.0-alpine` builder stage and copies it into a minimal `alpine:3.21` runtime image.

### Run

```bash
docker run -p 8080:8080 --env-file .env anonymizer
```

Copy `.env.example` to `.env` and adjust configuration before running.

### Docker Compose

The repository includes a `docker-compose.yml` that starts the anonymizer with a Redis cache backend:

```bash
docker compose up
```

This starts:
- **Redis 7 Alpine** on port `6379` (with health check)
- **Anonymizer** on port `8080` (with Redis caching enabled)

Verify the service is healthy:

```bash
curl http://localhost:8080/health
```

Stop:

```bash
docker compose down
```

## Kubernetes

### Minimal Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: anonymizer
spec:
  replicas: 1
  selector:
    matchLabels:
      app: anonymizer
  template:
    metadata:
      labels:
        app: anonymizer
    spec:
      containers:
        - name: anonymizer
          image: anonymizer:latest
          ports:
            - containerPort: 8080
          envFrom:
            - configMapRef:
                name: anonymizer-config
          readinessProbe:
            httpGet:
              path: /health
              port: 8080
            initialDelaySeconds: 5
            periodSeconds: 10
          livenessProbe:
            httpGet:
              path: /health
              port: 8080
            initialDelaySeconds: 15
            periodSeconds: 20
```

### ConfigMap Example

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: anonymizer-config
data:
  PORT: "8080"
  LOG_LEVEL: "INFO"
  MAX_BATCH_SIZE: "100"
  PRIVACY_CACHE_ENABLED: "true"
  PRIVACY_CACHE_REDIS_ADDR: "redis-service:6379"
```

### Service

```yaml
apiVersion: v1
kind: Service
metadata:
  name: anonymizer
spec:
  selector:
    app: anonymizer
  ports:
    - port: 8080
      targetPort: 8080
  type: ClusterIP
```

## Scaling Considerations

- **Stateless except for cache:** The anonymizer is stateless. All state lives in Redis when caching is enabled. Without Redis, each replica maintains its own independent cache.
- **Redis for multi-replica:** When running multiple replicas, configure `PRIVACY_CACHE_REDIS_ADDR` to share cached rule matching results across instances.
- **Memory:** Buffer pooling via `sync.Pool` reduces GC pressure. Each request allocates approximately 3x the input size for processing.
- **CPU:** Enable concurrency (`PRIVACY_CONCURRENCY_*`) for workloads with large text payloads where parallel token and rule evaluation improves throughput.

## Security

- **No hardcoded credentials:** All configuration comes from environment variables.
- **Redis authentication:** Use `REDIS_ANONYMIZER_SERVICE_V2_CACHE_TOKEN` to authenticate with password-protected Redis instances.
- **Input limits:** The leakspok library enforces input size limits (default 10MB) to prevent memory exhaustion.

## See Also

- [Getting Started](./getting-started.md) — quick start with Docker
- [Configuration Reference](./configuration.md) — all environment variables
- [Architecture](./architecture.md) — internal design and performance optimizations
