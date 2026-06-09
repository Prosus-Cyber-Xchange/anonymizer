# Error Reference

This is the canonical reference for all error codes returned by the anonymizer. Error responses are always JSON.

## Error Response Format

```json
{
  "code": "ERROR_CODE",
  "error": "Human-readable description"
}
```

## Error Codes

| HTTP Status | Code | Cause | Resolution |
|-------------|------|-------|------------|
| `400` | `INVALID_REQUEST` | Malformed JSON body or failed to read request body | Check that the request body is valid JSON |
| `400` | `INVALID_SETTINGS` | Settings validation failed — empty entities, missing redaction or mask, invalid exception operator, or unsupported entity type | Check entity names and ensure each entity has either `redaction` or `mask` defined |
| `400` | `BATCH_SIZE_EXCEEDED` | Number of batch items exceeds `MAX_BATCH_SIZE` (default: `100`) | Reduce batch size or increase `MAX_BATCH_SIZE` |
| `400` | `NO_RULES` | text/plain request without `X-Anonymize-Entities` header and no context-injected rules | Add `X-Anonymize-Entities` header or configure a plugin to inject rules |
| `415` | `UNSUPPORTED_MEDIA_TYPE` | Unsupported `Content-Type` on `/api/v1/anonymize`, or non-JSON `Content-Type` on `/api/v1/anonymize/batch` | Use `application/json` or `text/plain` |
| `500` | `ANONYMIZATION_FAILED` | Unexpected internal error during anonymization processing | Check server logs for details |

## See Also

- [POST /api/v1/anonymize](./anonymize.md) — per-endpoint error details
- [POST /api/v1/anonymize/batch](./batch.md) — per-endpoint error details
