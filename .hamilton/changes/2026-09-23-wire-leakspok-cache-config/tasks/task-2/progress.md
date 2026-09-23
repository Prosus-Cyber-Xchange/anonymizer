---
artifact: task-progress
change: 2026-09-23-wire-leakspok-cache-config
task: 2
status: done
updated: 2026-09-23
decision: accepted
---

# Task Progress: Task 2 — Add cache singleflight and TTL jitter env vars to config

## Attempt 1 — 2026-09-23

**Outcome**: done

**Created**: none
**Modified**: pkg/config/env.go, pkg/config/env_test.go, pkg/server/app_test.go
**Deleted**: none

**Red**: `go test -count=1 ./pkg/config` → build failed:
`envConfig.Privacy.CacheSingleflightEnabled` and `envConfig.Privacy.CacheTTLJitterPercentage`
undefined (fields did not exist in `config.EnvConfig.Privacy` yet).

**Green**: added `CacheSingleflightEnabled bool` (env `CACHE_SINGLEFLIGHT_ENABLED`, default false)
and `CacheTTLJitterPercentage float64` (env `CACHE_TTL_JITTER_PERCENTAGE`, default 0) to the
Privacy struct in `pkg/config/env.go` next to `CacheTTL`, and mirrored both fields with identical
tags into the anonymous Privacy struct type in `pkg/server/app_test.go`; verify commands pass.

**Verification**:
- `go test -count=1 ./pkg/config` → `ok github.com/Prosus-Cyber-Xchange/anonymizer/pkg/config 0.930s` (pass)
- `go test -count=1 -run '^$' ./pkg/server` → `ok github.com/Prosus-Cyber-Xchange/anonymizer/pkg/server 0.736s [no tests to run]`, exit 0
- `go test -count=1 -v -run 'TestLoadEnv_SingleflightAndJitter' ./pkg/config` →
  `TestLoadEnv_SingleflightAndJitter` PASS and `TestLoadEnv_SingleflightAndJitterDefaults` PASS

**Notes**: Explicit parsing test uses `t.Setenv`; defaults test uses `os.Unsetenv` with defer
cleanup, exactly as specified in plan.md Task 2. Struct fields and tags copied verbatim from the
plan snippets; gofmt applied for alignment. Defaults preserve v0.2.0 behavior (singleflight off,
no jitter).
