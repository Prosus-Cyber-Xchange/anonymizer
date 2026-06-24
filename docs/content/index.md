# Anonymizer

A high-performance PII anonymization service built for AI prompt workloads.

> An [iFood](https://ifood.com.br) open-source project by the AI Security team.

## What it does

Detects and anonymizes personally identifiable information (PII) — emails, CPF/CNPJ numbers, IP addresses, credit cards, phone numbers, and more — using byte-level processing for minimal latency and maximal throughput.

## Highlights

- **Inline privacy rules** — supply anonymization settings directly in the request body
- **Batch processing** — anonymize multiple texts in a single request
- **Redaction and masking** — replace PII entirely or partially mask it
- **Regex exceptions** — use regex patterns to exclude values from anonymization
- **Global exceptions** — configure server-level exception patterns that apply to every rule
- **Plugin system** — inject custom middleware at compile time
- **Embeddable library** — import as a Go package in any application

## Where to start

[Getting Started](getting-started.md){ .md-button }
[Entity Reference](entities.md){ .md-button }
[API Specification](openapi.yaml){ .md-button }