---
artifact: task-progress
change: 2026-09-23-wire-leakspok-cache-config
task: 3
status: done
updated: 2026-09-23
decision: accepted
---

# Task Progress: Task 3 — Wire singleflight and jitter fields into analyzer cache options

## Attempt 1 — 2026-09-23

- Date: 2026-09-23
- Outcome: done
- Change:
  - Added guard test `TestNewFromConfig_SingleflightAndJitterWired` to pkg/server/app_test.go, mirroring `TestNewFromConfig_ConcurrencyConfigWired`: builds the app via `setupRedis(t)` with `CacheSingleflightEnabled=true` and `CacheTTLJitterPercentage=0.15`, then asserts `NewFromConfig` succeeds and `Handler()` is non-nil (construction guard — wiring is unobservable from outside `AnonymizerServer`; flag behavior is leakspok's, proven upstream).
  - Wired the two new config fields into the `analyzer.CacheOptions` literal in `NewFromConfig` (pkg/server/app.go), after `DisableInMemoryCache`:
    - `TTLJitterPercentage: a.envConfig.Privacy.CacheTTLJitterPercentage`
    - `SingleflightEnabled: a.envConfig.Privacy.CacheSingleflightEnabled`
- Created: none
- Modified: pkg/server/app.go, pkg/server/app_test.go
- Deleted: none
- Verification:
  - `go vet ./...` → clean (exit 0)
  - `go test -count=1 -run 'TestNewFromConfig' ./pkg/server` → ok (15.352s; Docker via colima `DOCKER_HOST=unix:///Users/caio.cavalcante/.colima/default/docker.sock TESTCONTAINERS_RYUK_DISABLED=true`)
  - `go build ./...` → exit 0
  - `go test -short -race -count=1 ./...` → all packages pass (e2e, internal/handler, internal/monitoring, pkg/config, pkg/privacy, pkg/server ok; harmless macOS linker LC_DYSYMTAB warnings only)
- Notes: none
