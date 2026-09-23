---
artifact: task-progress
change: 2026-09-23-wire-leakspok-cache-config
task: 1
status: done
updated: 2026-09-23
decision: accepted
---

# Task Progress: Task 1 — Update leakspok to v0.3.0 and re-vendor

## Attempt 1

- Date: 2026-09-23
- Outcome: done
- Steps run:
  1. `go get github.com/Prosus-Cyber-Xchange/leakspok@v0.3.0` → exit 0; upgraded v0.2.0 => v0.3.0
  2. `go mod tidy` → exit 0; diff reviewed: go.mod/go.sum changes limited to leakspok v0.2.0 => v0.3.0; golang.org/x/sync stays at v0.20.0
  3. `go mod vendor` → exit 0; vendor/ regenerated: leakspok v0.3.0 sources (wholesale) + new vendor/golang.org/x/sync/singleflight package
  4. Verify (below) → passed
- Verification:
  - `go mod verify` → "all modules verified", exit 0
  - `go build ./...` → exit 0
  - `go test -short -race -count=1 ./...` → all packages pass (see test note below)
  - `go vet ./...` → clean, exit 0 (AGENTS.md build/typecheck)
- Created:
  - vendor/github.com/Prosus-Cyber-Xchange/leakspok/analyzer/singleflight_coalescer.go
  - vendor/golang.org/x/sync/singleflight/singleflight.go
- Modified:
  - go.mod (leakspok v0.2.0 => v0.3.0)
  - go.sum (leakspok v0.2.0 => v0.3.0 hashes)
  - vendor/modules.txt (leakspok v0.3.0; adds golang.org/x/sync/singleflight)
  - vendor/github.com/Prosus-Cyber-Xchange/leakspok/analyzer/cache/options.go
  - vendor/github.com/Prosus-Cyber-Xchange/leakspok/analyzer/cache/valkey.go
  - vendor/github.com/Prosus-Cyber-Xchange/leakspok/analyzer/concurrent_runner.go
  - vendor/github.com/Prosus-Cyber-Xchange/leakspok/analyzer/factory.go
  - vendor/github.com/Prosus-Cyber-Xchange/leakspok/analyzer/serial_runner.go
- Deleted: none
- Notes:
  - No source files outside vendor/ changed (git diff --name-only outside vendor/: none).
  - Acceptance confirmed: go.mod requires leakspok v0.3.0; vendor/modules.txt records leakspok v0.3.0 and lists golang.org/x/sync/singleflight; repo compiles; unit tests pass.
  - Test note: testcontainers-based pkg/server tests require Docker. The colima VM (2 CPU / 4 GiB) could not start the testcontainers ryuk reaper; with the standard env config `DOCKER_HOST=unix:///Users/caio.cavalcante/.colima/default/docker.sock TESTCONTAINERS_RYUK_DISABLED=true` the full suite passes (all packages ok, including pkg/server 22.3s).

