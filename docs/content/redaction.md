# Redaction Strategies

The anonymizer supports two strategies for hiding PII: **redaction** (full replacement) and **mask** (partial replacement). Each entity can independently choose its strategy.

## Redaction

Redaction replaces the entire matched value with a fixed placeholder string.

**Configuration:**

```json
{
  "redaction": {
    "replacement": "<EMAIL_REDACTED>"
  }
}
```

| Field | Required | Description |
|-------|----------|-------------|
| `replacement` | Yes | The string that replaces the detected value |

**Example:**

| Input | Settings | Output |
|-------|----------|--------|
| `john@example.com` | `replacement: "<EMAIL>"` | `<EMAIL>` |

## Mask

Mask replaces the first N characters of the matched value with a masking character, preserving the rest of the value.

**Configuration:**

```json
{
  "mask": {
    "replacement": "*",
    "maxLength": 4
  }
}
```

| Field | Required | Description |
|-------|----------|-------------|
| `replacement` | Yes | Character used for masking (typically a single character like `*` or `#`) |
| `maxLength` | Yes | Number of characters to mask (must be > 0) |

**Example:**

| Input | Settings | Output |
|-------|----------|--------|
| `123.456.789-09` | `replacement: "*"`, `maxLength: 4` | `****.456.789-09` |

## Precedence

If both `redaction` and `mask` are defined for the same entity, **redaction takes precedence**. The mask configuration is ignored.

## Per-Entity Configuration

Each entity in the `entities` array can independently choose redaction, mask, or both. This means you can redact emails while masking CPF numbers in the same request:

```json
{
  "entities": [
    { "name": "EMAIL", "redaction": { "replacement": "<EMAIL>" } },
    { "name": "CPF_NUMBER", "mask": { "replacement": "*", "maxLength": 3 } }
  ]
}
```

## See Also

- [Entity Reference](./entities.md) — supported entity types
- [POST /api/v1/anonymize](./anonymize.md) — endpoint documentation with full examples
