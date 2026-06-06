# Getting Started

The anonymizer is a high-performance PII anonymization service designed for large text payloads such as AI prompts. Byte-level processing, buffer pooling, and optional concurrency ensure low latency and minimal resource consumption.

## Prerequisites

- **Docker** (recommended) — for the quickest start with Redis caching
- **Go 1.22+** — for running the service directly

## Option 1: Docker (Recommended)

```bash
# Clone the repository
git clone https://github.com/Prosus-Cyber-Xchange/anonymizer.git
cd anonymizer

# Start Redis + anonymizer
docker compose up
```

Wait for both services to show as healthy, then verify:

```bash
curl http://localhost:8080/health
```

Stop when done:

```bash
docker compose down
```

## Option 2: Manual (Go)

```bash
# Clone the repository
git clone https://github.com/Prosus-Cyber-Xchange/anonymizer.git
cd anonymizer

# Configure environment
cp .env.example .env

# Run the service
go run cmd/server/main.go
```

Verify:

```bash
curl http://localhost:8080/health
```

## First Request — JSON

Anonymize an email and CPF number using inline privacy settings:

```bash
curl -X POST http://localhost:8080/api/v1/anonymize \
  -H "Content-Type: application/json" \
  -d '{
    "text": "Contact john@example.com, CPF: 123.456.789-09",
    "settings": {
      "entities": [
        { "name": "EMAIL", "redaction": { "replacement": "<EMAIL>" } },
        { "name": "CPF_NUMBER", "redaction": { "replacement": "<CPF>" } }
      ]
    }
  }'
```

Response:

```json
{
  "anonymized_text": "Contact <EMAIL>, CPF: <CPF>",
  "detected_entities": ["CPF_NUMBER", "EMAIL"],
  "anonymized_entities": ["CPF_NUMBER", "EMAIL"]
}
```

## First Request — text/plain

The text/plain mode is ideal for piping large text directly without JSON overhead:

```bash
curl -X POST http://localhost:8080/api/v1/anonymize \
  -H "Content-Type: text/plain" \
  -H "X-Anonymize-Entities: EMAIL" \
  -H "X-Anonymize-Placeholder: <REDACTED>" \
  -d "Contact john@example.com"
```

Response body:

```
Contact <REDACTED>
```

Response headers include `X-Anonymize-Detected-Entities: EMAIL` and `X-Anonymize-Anonymized-Entities: EMAIL`.

## First Batch Request

Anonymize multiple texts in one request:

```bash
curl -X POST http://localhost:8080/api/v1/anonymize/batch \
  -H "Content-Type: application/json" \
  -d '[
    {
      "text": "Email: john@example.com",
      "settings": { "entities": [{ "name": "EMAIL", "redaction": { "replacement": "<EMAIL>" } }] }
    },
    {
      "text": "CPF: 123.456.789-09",
      "settings": { "entities": [{ "name": "CPF_NUMBER", "redaction": { "replacement": "<CPF>" } }] }
    }
  ]'
```

## Using as a Library

The anonymizer can be embedded in any Go application:

```go
package main

import (
    "context"
    "log"

    "github.com/Prosus-Cyber-Xchange/anonymizer/pkg/server"
)

func main() {
    anonymizer, err := server.NewFromConfig(context.Background())
    if err != nil {
        log.Fatal(err)
    }

    // Embed in your existing server
    mux := http.NewServeMux()
    mux.Handle("/anon/", http.StripPrefix("/anon", anonymizer.Handler()))

    // Or run standalone
    anonymizer.ListenAndServe(context.Background())
}
```

## Next Steps

- [Content Negotiation](./content-negotiation.md) — JSON vs text/plain modes in detail
- [Entity Reference](./entities.md) — all supported PII types
- [Redaction Strategies](./redaction.md) — redact and mask configuration
- [Configuration Reference](./configuration.md) — all environment variables
- [Plugin Developer Guide](./plugins.md) — extend the service with custom middleware
- [Deployment Guide](./deployment.md) — production deployment with Docker and Kubernetes
