# POST /api/v1/anonymize

Anonymizes PII in a given text using privacy rules provided inline in the request body. No service configuration is required — the caller supplies the anonymization settings directly.

To anonymize multiple texts in a single request, see [`POST /api/v1/anonymize/batch`](./batch.md).

## Request

**Method:** `POST`
**Path:** `/api/v1/anonymize`
**Content-Type:** `application/json`

### Body

```json
{
  "text": "<string to anonymize>",
  "settings": {
    "entities": [
      {
        "name": "<ENTITY_TYPE>",
        "exceptions": [...],
        "redaction": { "replacement": "<PLACEHOLDER>" }
      }
    ]
  }
}
```

| Field | Type | Required | Description |
|---|---|---|---|
| `text` | string | yes | The text to anonymize |
| `settings` | object | yes | Privacy rules to apply |
| `settings.entities` | array | yes | List of entity configurations. At least one required |
| `settings.entities[].name` | string | yes | Entity type to detect and anonymize (see [Supported Entities](#supported-entities)) |
| `settings.entities[].exceptions` | array | no | Patterns to skip during anonymization (see [Exceptions](#exceptions)) |
| `settings.entities[].redaction` | object | no* | Redaction strategy: replaces the matched value with a fixed placeholder |
| `settings.entities[].redaction.replacement` | string | yes | Placeholder string used to replace the matched value |
| `settings.entities[].mask` | object | no* | Mask strategy: replaces the first N characters with a masking character |
| `settings.entities[].mask.replacement` | string | yes | Character used for masking |
| `settings.entities[].mask.maxLength` | integer | yes | Number of characters to mask |

*Each entity must define either `redaction` or `mask`. If both are provided, `redaction` takes precedence.

## Response

**Content-Type:** `application/json`

```json
{
  "anonymized_text": "<anonymized result>",
  "detected_entities": ["EMAIL", "CPF"],
  "anonymized_entities": ["EMAIL", "CPF"]
}
```

| Field | Type | Description |
|---|---|---|
| `anonymized_text` | string | The input text after anonymization |
| `detected_entities` | array of strings | Entity types found in the text |
| `anonymized_entities` | array of strings | Entity types that were actually anonymized |

Both entity lists are sorted alphabetically and will be empty arrays (`[]`) when nothing was detected.

## Error Responses

| HTTP Status | Error Code | Cause |
|---|---|---|
| `400 Bad Request` | `INVALID_REQUEST` | Request body is not valid JSON |
| `400 Bad Request` | `INVALID_SETTINGS` | Settings failed validation (e.g. missing entity name, no redaction/mask defined, invalid exception operator) |
| `500 Internal Server Error` | `ANONYMIZATION_FAILED` | Unexpected failure during anonymization |

## Supported Entities

| Name | Description |
|---|---|
| `EMAIL` | Email addresses |
| `CPF_NUMBER` | Brazilian individual taxpayer registry (CPF) numbers |
| `CNPJ_NUMBER` | Brazilian company registry (CNPJ) numbers |
| `IP_ADDRESS` / `IP` | IPv4 and IPv6 addresses |
| `IPV4` | IPv4 addresses only |
| `IPV6` | IPv6 addresses only |
| `CREDIT_CARD` | Credit card numbers |
| `PHONE` | Phone numbers |
| `LINK` / `URL` | URLs and hyperlinks |
| `SSN` | US Social Security Numbers |
| `ADDRESS` | Street addresses |
| `BANK_INFO` | Banking information (IBAN) |
| `UUID` | UUIDs and GUIDs |

Entity names are case-insensitive in the request (`email` and `EMAIL` are both accepted).

## Exceptions

Exceptions let you skip anonymization for specific values that match a pattern. A common use case is allowing internal or system email addresses through unchanged.

### Exception structure

```json
{
  "reason": "Human-readable explanation for the exception",
  "match": {
    "operator": "<OPERATOR>",
    "pattern": "<value to match against>"
  }
}
```

| Field | Type | Required | Description |
|---|---|---|---|
| `reason` | string | yes | Explanation of why this value should not be anonymized |
| `match.operator` | string | yes | Matching strategy (see table below) |
| `match.pattern` | string | yes | Value to match against |

### Match operators

| Operator | Description | Example pattern | Matches |
|---|---|---|---|
| `equal` | Exact byte-for-byte match | `system@ifood.com.br` | Only `system@ifood.com.br` |
| `ignoreCaseEqual` | Case-insensitive exact match | `noreply@ifood.com.br` | `noreply@ifood.com.br`, `NoReply@ifood.com.br`, etc. |
| `startsWith` | Value starts with the pattern | `admin@` | `admin@ifood.com.br`, `admin@corp.com`, etc. |
| `endsWith` | Value ends with the pattern | `@ifood.com.br` | Any address on the `ifood.com.br` domain |

## Examples

### Redact emails and mask CPF

```json
POST /api/v1/anonymize
Content-Type: application/json

{
  "text": "Customer john.doe@example.com has CPF 123.456.789-09",
  "settings": {
    "entities": [
      {
        "name": "EMAIL",
        "redaction": {
          "replacement": "<EMAIL_REDACTED>"
        }
      },
      {
        "name": "CPF_NUMBER",
        "mask": {
          "replacement": "*",
          "maxLength": 3
        }
      }
    ]
  }
}
```

```json
HTTP 200 OK

{
  "anonymized_text": "Customer <EMAIL_REDACTED> has CPF ***.456.789-09",
  "detected_entities": ["CPF_NUMBER", "EMAIL"],
  "anonymized_entities": ["CPF_NUMBER", "EMAIL"]
}
```

### Redact emails with domain exception

```json
POST /api/v1/anonymize
Content-Type: application/json

{
  "text": "Sent from system@ifood.com.br to customer@gmail.com",
  "settings": {
    "entities": [
      {
        "name": "EMAIL",
        "exceptions": [
          {
            "reason": "Internal iFood addresses are not PII",
            "match": {
              "operator": "endsWith",
              "pattern": "@ifood.com.br"
            }
          }
        ],
        "redaction": {
          "replacement": "<EMAIL_REDACTED>"
        }
      }
    ]
  }
}
```

```json
HTTP 200 OK

{
  "anonymized_text": "Sent from system@ifood.com.br to <EMAIL_REDACTED>",
  "detected_entities": ["EMAIL"],
  "anonymized_entities": ["EMAIL"]
}
```

### Multiple exceptions on the same entity

```json
{
  "name": "EMAIL",
  "exceptions": [
    {
      "reason": "iFood internal domain",
      "match": { "operator": "endsWith", "pattern": "@ifood.com.br" }
    },
    {
      "reason": "Known system sender",
      "match": { "operator": "equal", "pattern": "noreply@partner.com" }
    }
  ],
  "redaction": { "replacement": "<EMAIL_REDACTED>" }
}
```

### No PII detected

When the text contains no detectable PII, the response returns the original text unchanged with empty entity lists.

```json
{
  "anonymized_text": "No sensitive data here.",
  "detected_entities": [],
  "anonymized_entities": []
}
```
