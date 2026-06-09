# Entity Reference

This is the canonical reference for all PII entity types supported by the anonymizer. Entity names are **case-insensitive** in requests (`email`, `Email`, and `EMAIL` are all accepted).

## Supported Entities

| Name | Aliases | Description | Example Detections |
|------|---------|-------------|-------------------|
| `EMAIL` | — | Email addresses | `user@example.com` |
| `CPF_NUMBER` | — | Brazilian CPF with check-digit validation | `123.456.789-09`, `12345678909` |
| `CNPJ_NUMBER` | — | Brazilian CNPJ with check-digit validation | `12.345.678/0001-90` |
| `IP_ADDRESS` | `IP` | IPv4 and IPv6 addresses | `192.168.1.1`, `::1` |
| `IPV4` | — | IPv4 addresses only | `192.168.1.1` |
| `IPV6` | — | IPv6 addresses only | `::1`, `2001:db8::1` |
| `CREDIT_CARD` | — | Credit card numbers | `4111-1111-1111-1111` |
| `PHONE` | — | Phone numbers, international and Brazilian formats | `+55 11 99999-9999` |
| `LINK` | `URL` | URLs and hyperlinks | `https://example.com` |
| `SSN` | — | US Social Security Numbers | `123-45-6789` |
| `ADDRESS` | — | Street addresses | `123 Main St, Springfield` |
| `BANK_INFO` | — | Banking information including IBAN | `DE89 3704 0044` |
| `UUID` | — | UUIDs and GUIDs | `550e8400-e29b-41d4-a716-446655440000` |

## Custom Entities

Adding new entity types requires changes to the [leakspok](https://github.com/Prosus-Cyber-Xchange/leakspok) library's `pattern` package. Custom matchers can be registered in leakspok and then mapped in the anonymizer's rule builder.

## See Also

- [Redaction Strategies](./redaction.md) — how to configure anonymization per entity
- [POST /api/v1/anonymize](./anonymize.md) — endpoint documentation with request examples
