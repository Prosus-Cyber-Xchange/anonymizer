# Anonymizer

A high-performance PII anonymization service built for AI prompt workloads.

## What it does

Detects and anonymizes personally identifiable information (PII) — emails, CPF/CNPJ numbers, IP addresses, credit cards, phone numbers, and more — using byte-level processing for minimal latency and maximal throughput.

## Highlights

- **Inline privacy rules** — supply anonymization settings directly in the request body
- **Batch processing** — anonymize multiple texts in a single request
- **Redaction and masking** — replace PII entirely or partially mask it
- **Plugin system** — inject custom middleware at compile time
- **Embeddable library** — import as a Go package in any application

## Where to start

[Getting Started](getting-started.md){ .md-button }
[Entity Reference](entities.md){ .md-button }
[API Specification](openapi.yaml){ .md-button }