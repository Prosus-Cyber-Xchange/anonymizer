---
artifact: task-feedback
change: 2026-09-23-wire-leakspok-cache-config
task: 4
status: approved
updated: 2026-09-23
decision: accepted
---
Base: 0f836f63b8552355cad7295d7354a988b39dfde8
Head: 95cb7257aa5879d3376d7a636cd93cc49a668d3d
Verdict: approved
### Blocking
- none
### Suggestions
- docs/content/configuration.md:29-30 — Both new rows verified accurate against pkg/config/env.go:17-18 (CACHE_SINGLEFLIGHT_ENABLED default false, CACHE_TTL_JITTER_PERCENTAGE default 0, PRIVACY_ prefix) and vendored leakspok v0.3.0 semantics (singleflight gated on Cache.Enabled && SingleflightEnabled in concurrent_runner.go:32 / serial_runner.go:25; TTLJitterPercentage fraction, <= 0 disables jitter in cache/options.go:13-14 and valkey.go:93).
