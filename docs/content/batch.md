# POST /api/v1/anonymize/batch

Anonymizes PII in multiple texts in a single request. Each item is processed independently using its own inline privacy settings. The response preserves the order of the input array.

This endpoint fails fast: if any item fails validation or anonymization, the entire request returns an error and no results are returned.

## Request

**Method:** `POST`
**Path:** `/api/v1/anonymize/batch`
**Content-Type:** `application/json`

### Body

An array of anonymization requests. Each item follows the same structure as [`POST /api/v1/anonymize`](./anonymize.md).

```json
[
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
]
```

| Field | Type | Required | Description |
|---|---|---|---|
| `[].text` | string | yes | The text to anonymize |
| `[].settings` | object | yes | Privacy rules to apply to this item |
| `[].settings.entities` | array | yes | List of entity configurations. At least one required |
| `[].settings.entities[].name` | string | yes | Entity type to detect and anonymize (see [Supported Entities](#supported-entities)) |
| `[].settings.entities[].exceptions` | array | no | Patterns to skip during anonymization (see [Exceptions](#exceptions)) |
| `[].settings.entities[].redaction` | object | no* | Redaction strategy: replaces the matched value with a fixed placeholder |
| `[].settings.entities[].redaction.replacement` | string | yes | Placeholder string used to replace the matched value |
| `[].settings.entities[].mask` | object | no* | Mask strategy: replaces the first N characters with a masking character |
| `[].settings.entities[].mask.replacement` | string | yes | Character used for masking |
| `[].settings.entities[].mask.maxLength` | integer | yes | Number of characters to mask |

*Each entity must define either `redaction` or `mask`. If both are provided, `redaction` takes precedence.

### Limits

The maximum number of items per request is controlled by the `MAX_BATCH_SIZE` environment variable (default: `100`). Requests exceeding this limit are rejected with `400 BATCH_SIZE_EXCEEDED`.

## Response

**Content-Type:** `application/json`

An array of anonymization results in the same order as the input.

```json
[
  {
    "anonymized_text": "<anonymized result>",
    "detected_entities": ["EMAIL", "CPF_NUMBER"],
    "anonymized_entities": ["EMAIL", "CPF_NUMBER"]
  }
]
```

| Field | Type | Description |
|---|---|---|
| `[].anonymized_text` | string | The input text after anonymization |
| `[].detected_entities` | array of strings | Entity types found in this item's text |
| `[].anonymized_entities` | array of strings | Entity types that were actually anonymized in this item |

Both entity lists are sorted alphabetically and will be empty arrays (`[]`) when nothing was detected.

## Error Responses

For the complete error code reference, see [errors.md](./errors.md).

## Supported Entities

For the full entity reference, see [entities.md](./entities.md).

## Exceptions

Each item can define exceptions per entity to skip anonymization for specific matching values. See the [Exceptions section in the anonymize endpoint docs](./anonymize.md#exceptions) for the full reference.

## Examples

### Redact different entities per item

```json
POST /api/v1/anonymize/batch
Content-Type: application/json

[
  {
    "text": "Customer john.doe@example.com has CPF 123.456.789-09",
    "settings": {
      "entities": [
        {
          "name": "EMAIL",
          "redaction": { "replacement": "<EMAIL_REDACTED>" }
        },
        {
          "name": "CPF_NUMBER",
          "mask": { "replacement": "*", "maxLength": 3 }
        }
      ]
    }
  },
  {
    "text": "Call me at +55 11 99999-9999",
    "settings": {
      "entities": [
        {
          "name": "PHONE",
          "redaction": { "replacement": "<PHONE_REDACTED>" }
        }
      ]
    }
  }
]
```

```json
HTTP 200 OK

[
  {
    "anonymized_text": "Customer <EMAIL_REDACTED> has CPF ***.456.789-09",
    "detected_entities": ["CPF_NUMBER", "EMAIL"],
    "anonymized_entities": ["CPF_NUMBER", "EMAIL"]
  },
  {
    "anonymized_text": "Call me at <PHONE_REDACTED>",
    "detected_entities": ["PHONE"],
    "anonymized_entities": ["PHONE"]
  }
]
```

### With exceptions on a batch item

```json
POST /api/v1/anonymize/batch
Content-Type: application/json

[
  {
    "text": "Sent from system@ifood.com.br to customer@gmail.com",
    "settings": {
      "entities": [
        {
          "name": "EMAIL",
          "exceptions": [
            {
              "reason": "Internal iFood addresses are not PII",
              "match": { "operator": "endsWith", "pattern": "@ifood.com.br" }
            }
          ],
          "redaction": { "replacement": "<EMAIL_REDACTED>" }
        }
      ]
    }
  }
]
```

```json
HTTP 200 OK

[
  {
    "anonymized_text": "Sent from system@ifood.com.br to <EMAIL_REDACTED>",
    "detected_entities": ["EMAIL"],
    "anonymized_entities": ["EMAIL"]
  }
]
```

### Batch size exceeded

```json
HTTP 400 Bad Request

{
  "code": "BATCH_SIZE_EXCEEDED",
  "message": "batch size 150 exceeds maximum allowed size of 100"
}
```

### No PII detected in an item

When an item's text contains no detectable PII, its result returns the original text unchanged with empty entity lists.

```json
[
  {
    "anonymized_text": "No sensitive data here.",
    "detected_entities": [],
    "anonymized_entities": []
  }
]
```
