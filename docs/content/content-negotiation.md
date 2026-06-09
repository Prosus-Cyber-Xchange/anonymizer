# Content Negotiation

The `/api/v1/anonymize` endpoint accepts two content types, letting you choose between structured JSON requests and lightweight header-based text/plain requests.

## Overview

| Content-Type | Use Case | Best For |
|-------------|----------|----------|
| `application/json` | Inline privacy settings in the request body | Complex settings with exceptions per entity |
| `text/plain` | Rules via HTTP headers, raw text body | Simple rules, piping large text without JSON overhead |

## JSON Mode (`application/json`)

Rules are provided inline in the request body under the `settings` field. Each entity can have its own redaction or mask strategy, plus per-entity exceptions.

See [POST /api/v1/anonymize](./anonymize.md) for the full request and response format.

## text/plain Mode (`text/plain`)

The request body is the raw text to anonymize. Anonymization rules are configured via HTTP headers.

### Request Headers

| Header | Required | Default | Description |
|--------|----------|---------|-------------|
| `X-Anonymize-Entities` | Yes | — | Comma-separated entity names: `EMAIL,CPF_NUMBER,CREDIT_CARD` |
| `X-Anonymize-Strategy` | No | `redact` | Anonymization strategy: `redact` or `mask` |
| `X-Anonymize-Placeholder` | No | `<REDACTED>` | Replacement string (redact mode only) |
| `X-Anonymize-Mask-Char` | No | `*` | Masking character (mask mode only) |
| `X-Anonymize-Mask-Length` | No | `4` | Characters to mask (mask mode only) |

All entities in `X-Anonymize-Entities` share the same strategy and placeholder/mask settings. If you need different strategies per entity, use JSON mode.

### Response

- **Body:** The anonymized text as raw text (`Content-Type: text/plain; charset=utf-8`)
- **Headers:**

| Header | Description |
|--------|-------------|
| `X-Anonymize-Detected-Entities` | Comma-separated list of detected entity types |
| `X-Anonymize-Anonymized-Entities` | Comma-separated list of anonymized entity types |

### Example

```bash
curl -X POST http://localhost:8080/api/v1/anonymize \
  -H "Content-Type: text/plain" \
  -H "X-Anonymize-Entities: EMAIL,CPF_NUMBER" \
  -H "X-Anonymize-Placeholder: <REDACTED>" \
  -d "Contact john@example.com, CPF 123.456.789-09"
```

Response body:
```
Contact <REDACTED>, CPF <REDACTED>
```

Response headers:
```
Content-Type: text/plain; charset=utf-8
X-Anonymize-Detected-Entities: CPF_NUMBER,EMAIL
X-Anonymize-Anonymized-Entities: CPF_NUMBER,EMAIL
```

## Default Behavior

When no `Content-Type` header is provided, the request is treated as `application/json`.

## Unsupported Content Types

Any `Content-Type` other than `application/json` or `text/plain` returns:

```json
HTTP 415 Unsupported Media Type
{
  "code": "UNSUPPORTED_MEDIA_TYPE",
  "error": "unsupported content type: application/xml"
}
```

## Batch Endpoint

`/api/v1/anonymize/batch` accepts **only** `application/json`. Non-JSON content types return a `415 Unsupported Media Type` error.

## Rule Precedence (text/plain mode)

When a text/plain request arrives, the handler resolves rules in this order:

1. **`X-Anonymize-Entities` header** (highest priority) — if present, the header's entities are used
2. **Context-injected rules** (from plugin middleware) — used when no `X-Anonymize-Entities` header is present
3. **400 `NO_RULES` error** — returned when neither headers nor context rules are available

This allows plugins to set default rules for a service while still letting clients override them per request via headers. See [Plugin Developer Guide](./plugins.md) for details.

## See Also

- [POST /api/v1/anonymize](./anonymize.md) — full JSON endpoint documentation
- [Plugin Developer Guide](./plugins.md) — injecting rules via middleware
