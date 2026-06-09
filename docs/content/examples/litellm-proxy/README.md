# LiteLLM Proxy + Anonymizer PII Redaction

This example shows how to configure a [LiteLLM](https://docs.litellm.ai) proxy with a custom middleware that redacts PII (Personally Identifiable Information) before sending requests to an LLM, and restores the original values in the response.

> **Files in this folder:**
>
> | File | Purpose |
> |------|---------|
> | `litellm/pii_middleware.py` | LiteLLM callback — anonymizes before call, restores after response |
> | `litellm/config.yaml` | LiteLLM proxy configuration (model, callbacks, keys) |
> | `litellm/Dockerfile` | Container image for the LiteLLM proxy with custom middleware |
> | `litellm/requirements.txt` | Python dependencies (`httpx`) |
> | `docker-compose.yml` | Orchestrates Redis, Anonymizer, and LiteLLM Proxy |
> | `.env.example` | Template for your OpenAI API key |
> | `curls/test.sh` | End-to-end test script (3 scenarios) |

## Architecture

```
User curl              LiteLLM Proxy          Anonymizer         OpenAI API
  │                        │                      │                   │
  ├─POST /chat/completions─►                      │                   │
  │  "my email is          │                      │                   │
  │   bob@example.com"     │                      │                   │
  │                        ├─async_pre_call_hook──►                   │
  │                        │  POST /api/v1/anonymize                  │
  │                        │  "bob@example.com"──►│                   │
  │                        │◄─"[EMAIL]"───────────│                   │
  │                        │                      │                   │
  │                        ├─POST /chat/completions (anonymized)──────►
  │                        │  "my email is [EMAIL]"                   │
  │                        │◄─ "I see your email is [EMAIL]"──────────│
  │                        │                      │                   │
  │                        ├─async_post_call_success_hook              │
  │                        │  [EMAIL] → bob@example.com               │
  │◄─ "I see your email    │                      │                   │
  │    is bob@example.com"  │                      │                   │
```

## Step-by-step walkthrough

### Step 1 — Prerequisites

You need:

- **Docker and Docker Compose** installed
- **An OpenAI API key** (or any LiteLLM-compatible provider)

### Step 2 — Set your API key

```bash
cd docs/examples/litellm-proxy

cp .env.example .env
```

Edit `.env` and replace the placeholder:

```env
OPENAI_API_KEY=sk-YourRealKeyHere
```

### Step 3 — Start all services

```bash
docker compose up -d --build
```

Three containers start:

| Container | Port | What it does |
|-----------|------|-------------|
| `redis` | 6379 | Cache backend so the anonymizer can skip re-processing seen PII |
| `anonymizer` | 8080 | The PII detection/redaction API — built from the repo root Dockerfile |
| `litellm` | 4000 | OpenAI-compatible proxy with the PII middleware loaded |

Wait ~30 seconds for health checks to pass. `docker compose ps` should show all three as `healthy`.

### Step 4 — Run the tests

```bash
sh curls/test.sh
```

The script waits for both services, then runs three scenarios:

#### Test 1: PII redaction in user messages

```bash
curl -s http://localhost:4000/chat/completions \
  -H "Authorization: Bearer sk-local-demo-key" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o",
    "max_tokens": 60,
    "messages": [{"role": "user", "content": "My email is bob@example.com and phone is +55 11 99999-9999. What is prompt injection?"}]
  }'
```

**Verifies**: The LLM response does NOT contain `bob@example.com` — it was redacted before reaching the model.

#### Test 2: De-anonymization restores PII

The LLM is asked to echo the user's text. The response comes back with placeholders like `[EMAIL]`. The middleware swaps them back to the original values before returning to the caller.

**Verifies**: The response DOES contain `alice@foo.com` (not `[EMAIL]`).

#### Test 3: Benign text passes through

A request with no PII flows through untouched — no anonymizer call, no mapping, no de-anonymization.

**Verifies**: The response contains normal text and no errors occur.

### Step 5 — Send your own requests

Once tests pass, you can call the proxy directly as a drop-in replacement for the OpenAI API:

```bash
curl -s http://localhost:4000/chat/completions \
  -H "Authorization: Bearer sk-local-demo-key" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o",
    "messages": [{"role": "user", "content": "Can you summarize this? Contact: alice@foo.com, CPF 529.982.247-25"}]
  }' | python3 -c "import sys,json; print(json.load(sys.stdin)['choices'][0]['message']['content'])"
```

The LLM will see the anonymized version (`Contact: [EMAIL], CPF [CPF]`), but your response will have the original values restored.

### Step 6 — Stop the services

```bash
docker compose down
```

## How the middleware works

The middleware is a LiteLLM [Custom Callback](https://docs.litellm.ai/docs/proxy/call_hooks) implementing two hooks:

### `async_pre_call_hook` — before the LLM call

1. Loops over every `message` in the chat completion request.
2. For each message, calls `POST /api/v1/anonymize` with settings for EMAIL, PHONE, CPF_NUMBER, CREDIT_CARD, and IP_ADDRESS.
3. Uses Python's `difflib.SequenceMatcher` to find which parts of the original text were replaced (the placeholder strings like `[EMAIL]`) and their original values.
4. Stores the mapping (`"[EMAIL]" → "bob@example.com"`) keyed by a random request ID.
5. Replaces the message content with the anonymized version before LiteLLM forwards it to the model.

### `async_post_call_success_hook` — after the LLM call

1. Looks up the stored mapping by the request ID.
2. Scans the LLM's response for any placeholder strings.
3. Replaces each with the original PII value.
4. The end user sees the response with PII restored while the LLM never saw it.

### PII types redacted

| Entity | Placeholder | Example |
|--------|-------------|---------|
| EMAIL | `[EMAIL]` | `bob@example.com` → `[EMAIL]` |
| PHONE | `[PHONE]` | `+55 11 99999-9999` → `[PHONE]` |
| CPF_NUMBER | `[CPF]` | `529.982.247-25` → `[CPF]` |
| CREDIT_CARD | `[CC]` | `4111-1111-1111-1111` → `[CC]` |
| IP_ADDRESS | `[IP]` | `192.168.1.1` → `[IP]` |

## Customizing the example

### Adding entity types

Edit `PII_ENTITIES` in `litellm/pii_middleware.py`:

```python
PII_ENTITIES = [
    {"name": "EMAIL",    "redaction": {"replacement": "[EMAIL]"}},
    {"name": "SSN",      "redaction": {"replacement": "[SSN]"}},      # added
    {"name": "LINK",     "redaction": {"replacement": "[URL]"}},      # added
]
```

Also update the `PLACEHOLDER_RE` regex at the top of the file so the de-anonymization step can find the new placeholders:

```python
PLACEHOLDER_RE = re.compile(r"\[(?:EMAIL|PHONE|CPF|CC|IP|SSN|URL)\]")
```

See [entities.md](../../entities.md) for every supported entity type.

### Switching the LLM provider

Edit `model_list` in `litellm/config.yaml` — LiteLLM supports [100+ providers](https://docs.litellm.ai/docs/providers):

```yaml
model_list:
  - model_name: claude-sonnet
    litellm_params:
      model: anthropic/claude-sonnet-4-20250514
      api_key: os.environ/ANTHROPIC_API_KEY
```

Set the corresponding key in `.env`:

```env
ANTHROPIC_API_KEY=sk-ant-YourKeyHere
```

### Using Azure OpenAI

```yaml
model_list:
  - model_name: gpt-4o
    litellm_params:
      model: azure/gpt-4o
      api_key: os.environ/AZURE_API_KEY
      api_base: os.environ/AZURE_API_BASE
```

```env
AZURE_API_KEY=...
AZURE_API_BASE=https://your-resource.openai.azure.com
```

### Adding exception patterns

To skip redacting certain values (e.g., internal email domains):

```python
{"name": "EMAIL",
 "exceptions": [{"reason": "internal", "match": {"operator": "endsWith", "pattern": "@mycorp.com"}}],
 "redaction": {"replacement": "[EMAIL]"}},
```

Emails matching `*@mycorp.com` will pass through the anonymizer unchanged.

## Limitations

- **Streaming responses** (`stream: true`) are not supported — de-anonymization only runs in `async_post_call_success_hook`, which fires after the full response. Use non-streaming.
- **Multiple PII of the same type** in a single message can cause positional ambiguity during de-anonymization when the LLM echoes back placeholders out of order.
- **Nested or array content** in messages is not handled — only top-level string `content` fields are processed.
