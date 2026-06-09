# LiteLLM Proxy + Anonymizer PII Redaction

This example shows how to configure a [LiteLLM](https://docs.litellm.ai) proxy with a custom middleware that calls the Anonymizer service to redact PII (Personally Identifiable Information) before sending requests to an LLM, and restores original values in the response.

## Architecture

```
User curl              LiteLLM Proxy          Anonymizer         OpenAI API
  │                        │                      │                   │
  ├─POST /chat/complet────►│                      │                   │
  │  "my email is          │                      │                   │
  │   bob@example.com"     │                      │                   │
  │                        ├─async_pre_call_hook──►                   │
  │                        │  POST /api/v1/anonymize                  │
  │                        │  "bob@example.com"──►│                   │
  │                        │◄─"[EMAIL]"───────────│                   │
  │                        │                      │                   │
  │                        ├─POST /chat/complet (anonymized)──────────►
  │                        │  "my email is [EMAIL]"                   │
  │                        │◄─ "I see your email is [EMAIL]"──────────│
  │                        │                      │                   │
  │                        ├─async_post_call_success_hook              │
  │                        │  [EMAIL] → bob@example.com               │
  │◄─ "I see your email    │                      │                   │
  │    is bob@example.com"  │                      │                   │
```

## Quick Start

### Prerequisites

- Docker and Docker Compose
- An OpenAI API key (set via `.env`)

### Setup

```bash
cd docs/examples/litellm-proxy

# Copy and set your OpenAI API key
cp .env.example .env
# Edit .env with your actual key: OPENAI_API_KEY=sk-...

# Start all services
docker compose up -d --build

# Wait for services to be healthy, then run tests
sh curls/test.sh
```

### Services

| Service | Port | Description |
|---------|------|-------------|
| Redis | 6379 | Cache backend for the anonymizer |
| Anonymizer | 8080 | PII detection and redaction API |
| LiteLLM Proxy | 4000 | OpenAI-compatible proxy with PII middleware |

## How It Works

### Middleware (`pii_middleware.py`)

The middleware is a LiteLLM [Custom Callback](https://docs.litellm.ai/docs/proxy/call_hooks) that implements two hooks:

**`async_pre_call_hook`** — runs before every LLM request:
1. Iterates over all messages in the chat completion request
2. For each message, calls `POST /api/v1/anonymize` with settings for EMAIL, PHONE, CPF_NUMBER, CREDIT_CARD, and IP_ADDRESS
3. Stores the mapping of `placeholder → original value` (using `difflib` to diff original vs anonymized text)
4. Replaces the message content with the anonymized version

**`async_post_call_success_hook`** — runs after a successful (non-streaming) LLM response:
1. Retrieves the stored `placeholder → original` mapping for this request
2. Scans the LLM response for each placeholder string (e.g., `[EMAIL]`)
3. Replaces it with the original PII value

### PII Types Redacted

| Entity | Placeholder | Example |
|--------|-------------|---------|
| EMAIL | `[EMAIL]` | `bob@example.com` → `[EMAIL]` |
| PHONE | `[PHONE]` | `+55 11 99999-9999` → `[PHONE]` |
| CPF_NUMBER | `[CPF]` | `529.982.247-25` → `[CPF]` |
| CREDIT_CARD | `[CC]` | `4111-1111-1111-1111` → `[CC]` |
| IP_ADDRESS | `[IP]` | `192.168.1.1` → `[IP]` |

## Testing

The `curls/test.sh` script runs three scenarios:

1. **PII redaction** — Sends text containing an email and phone number; verifies the LLM never sees the raw PII
2. **De-anonymization** — The LLM echoes back a placeholder; verifies the response restores the original PII
3. **No PII pass-through** — Sends benign text; verifies it flows through unchanged

## Customization

### Adding entity types

Edit `PII_ENTITIES` in `pii_middleware.py`:

```python
PII_ENTITIES = [
    {"name": "EMAIL", "redaction": {"replacement": "[EMAIL]"}},
    {"name": "SSN", "redaction": {"replacement": "[SSN]"}},        # add
    {"name": "LINK", "redaction": {"replacement": "[URL]"}},       # add
]
```

See [entities.md](../../entities.md) for the full list of supported entity types.

### Changing the LLM model

Edit `model_list` in `config.yaml`:

```yaml
model_list:
  - model_name: gpt-4o
    litellm_params:
      model: azure/gpt-4o           # use Azure deployment
      api_key: os.environ/AZURE_API_KEY
      api_base: os.environ/AZURE_API_BASE
```

### Exception patterns

To skip redacting certain values (e.g., internal email domains), add exceptions to the anonymizer settings:

```python
{"name": "EMAIL",
 "exceptions": [{"reason": "internal", "match": {"operator": "endsWith", "pattern": "@mycorp.com"}}],
 "redaction": {"replacement": "[EMAIL]"}},
```

## Limitations

- **Streaming responses** are not supported for de-anonymization (use non-streaming, `stream: false`)
- **Multiple PII of the same type** within a single message may cause positional ambiguity in de-anonymization mapping
- **Nested or array content** in messages is not handled (only top-level `content` as a string)
