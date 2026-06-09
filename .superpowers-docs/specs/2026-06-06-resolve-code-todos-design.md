# Resolve Code TODOs: Metrics, Span Names, Dead Code, Logger, Concurrency

**Date:** 2026-06-06
**Scope:** `anonymizer-service-v2-ce/` — resolve 6 codebase TODOs

---

## Decisions Made

| Decision | Choice |
|----------|--------|
| Metrics reporting scope | Duration histogram + entity counter; skip body size; skip on errors |
| Span rename confirmations | Rename both handler spans; batch span stays |
| rulesLoader removal scope | Entire chain: field, param, deprecated `Anonymize()`, `LoadRuleSet`, `filterRules`, interface, mock |
| newLogger placement | Private function in `pkg/anonymizer/app.go` |
| Concurrency field names | Uncomment block as-is, no field name changes |
| Concurrency test | Keep existing test as-is |

---

## 1. Wire Metric Reporting

### Current state
`PrivacyMetrics` (3 methods: `ObserveRequestBodySize`, `ObserveAnonymizationDuration`, `CountAnonymizedEntity`) is stored on `Handler` but never called.

### Change
In each of the three handler methods (`anonymizeJSON`, `anonymizeTextPlain`, `AnonymizeBatch`):

- Wrap the `AnonymizeWithRules` call with `time.Since(start)` and report via `h.metrics.ObserveAnonymizationDuration(duration)`
- After a successful anonymize call, iterate over `output.Details.AnonymizedEntities` and call `h.metrics.CountAnonymizedEntity(entity)` for each
- Skip metric reporting on error paths (no duration, no entity counts)

`ObserveRequestBodySize` remains unused per design decision.

### Files
- `internal/handler/handler.go`

---

## 2–3. Rename Span Operation Names

| Current | New | Location |
|---------|-----|----------|
| `"anonymize_general_request"` | `"anonymize_json_request"` | `handler.go:103` |
| `"anonymize_text_plain"` | `"anonymize_textplain_request"` | `handler.go:157` |

No other files reference these strings. No test updates needed.

### Files
- `internal/handler/handler.go`

---

## 4. Remove rulesLoader Chain

### Current state
`rulesLoader` field on `Service` is always `nil` in production. The only codepath that uses it is the deprecated `Anonymize()` method → `LoadRuleSet()` → `rulesLoader.Load()`. Rules in production come from inline settings or context injection, not from a loader.

### What to remove
- `rulesLoader` field from `Service` struct
- `rulesLoader` parameter from `NewService()`
- Deprecated `Anonymize()` method
- `LoadRuleSet()` method
- `filterRules()` function
- `PrivacyRulesLoader` interface declaration and its `go:generate` directive
- `pkg/privacy/mocks/mock_rules_loader.go` (generated mock)

### What to update
- `pkg/anonymizer/app.go:117` — drop the `nil` argument: `NewService(*a.byteAnalyzer, a.logger)`
- `pkg/privacy/service_test.go` — remove all `rulesLoader` mock arguments from `NewService` calls; delete test cases that test `LoadRuleSet`
- `internal/handler/handler_test.go` — remove `rulesLoader` mock arguments from `NewService` calls
- `internal/handler/handler_text_test.go` — same
- `internal/handler/handler_metrics_test.go` — same

### Files
- `pkg/privacy/service.go`
- `pkg/privacy/service_test.go`
- `pkg/privacy/mocks/mock_rules_loader.go` (delete)
- `pkg/anonymizer/app.go`
- `internal/handler/handler_test.go`
- `internal/handler/handler_text_test.go`
- `internal/handler/handler_metrics_test.go`

---

## 5. Extract newLogger Function

### Change
Extract the inline logger creation from `NewFromConfig` into a private function:

```go
func newLogger(envConfig config.EnvConfig) *slog.Logger {
    var level slog.Level
    switch strings.ToUpper(envConfig.LogLevel) {
    case "DEBUG":
        level = slog.LevelDebug
    case "WARN", "WARNING":
        level = slog.LevelWarn
    case "ERROR":
        level = slog.LevelError
    default:
        level = slog.LevelInfo
    }
    return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
}
```

Replace the inline block at lines 56-69 with `a.logger = newLogger(envConfig)`.

Behavior is identical. No test updates needed.

### Files
- `pkg/anonymizer/app.go`

---

## 6. Wire Concurrency Options

### Current state
All 6 concurrency env vars are defined and parsed in `pkg/config/env.go`. Leakspok's `RunnerOptions.Concurrency` field and `MakeByteAnalyzer` fully support them. The wiring is commented out.

### Change
Uncomment the wiring block in `app.go`. The commented-out code uses field names that don't match leakspok's `ConcurrencyOptions` struct — the assignment must use the correct struct field names (`ConcurrentTokenProcessing`, `ConcurrentRuleProcessing`) to compile:

```go
runnerOpts.Concurrency = analyzer.ConcurrencyOptions{
    Enabled:                   envConfig.Privacy.ConcurrencyEnabled,
    ConcurrentTokenProcessing: envConfig.Privacy.ConcurrencyTokenProcessing,
    ConcurrentRuleProcessing:  envConfig.Privacy.ConcurrencyRuleProcessing,
    RuleRunnerPoolSize:        envConfig.Privacy.ConcurrencyRuleRunnerPoolSize,
    TokenPoolSize:             envConfig.Privacy.ConcurrencyTokenPoolSize,
    MaxGoroutineIdleTimeout:   envConfig.Privacy.ConcurrencyMaxGoroutineIdleTimeout,
}
```

Remove the TODO comment above it.

### Test
Keep `TestNewFromConfig_ConcurrencyConfigWired` as-is (verifies config parsing succeeds).

### Files
- `pkg/anonymizer/app.go`

---

## Verification

- `task test` — all unit and e2e tests pass
- `task lint` — no lint violations
- `task format` — code properly formatted
- `task build` — binary compiles