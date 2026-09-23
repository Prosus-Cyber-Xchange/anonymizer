---
artifact: review
change: 2026-09-23-wire-leakspok-cache-config
created: 2026-09-23
status: complete
decision: accepted
---

# Whole-branch Review: Wire leakspok cache config (singleflight + TTL jitter)

## Pass 1 — 2026-09-23

Base: f2b5c5ecd36726311ddbdfe414086864db55e4f0
Head: af6552780410f29120a3a8cd85e6342ab5ee2d4a
Verdict: approved

### Blocking

- None.

### Suggestions

- [.env.example] The two new cache vars (PRIVACY_CACHE_SINGLEFLIGHT_ENABLED, PRIVACY_CACHE_TTL_JITTER_PERCENTAGE) are not listed in the sample env file. Optional: add them next to PRIVACY_CACHE_TTL for operator discoverability; plan.md scoped docs rows to docs/content/configuration.md only, so this is a non-blocking convenience improvement.
- [docs/content/configuration.md:41] Pre-existing row `REDIS_ANONYMIZER_SERVICE_V2_CACHE_TOKEN` does not match the actual env name `PRIVACY_REDIS_CACHE_TOKEN` (env.go:33 RedisToken tag `REDIS_CACHE_TOKEN` under envPrefix `PRIVACY_`). Predates this branch; not part of Task 4 scope. Optional consistency fix in a follow-up.
- [tasks/task-2/progress.md:3] Frontmatter `status: pending` contradicts the body "Outcome: done" and the root progress.md done status for Task 2. Bookkeeping hygiene only; does not affect the approved task evidence.

## Pass 2 — 2026-09-23

Base: f2b5c5ecd36726311ddbdfe414086864db55e4f0
Head: 0e0a076a49dc16db4388196237f0b741090b0aa4
Verdict: approved

### Blocking

- None.

### Suggestions

- Resolved: prior-pass suggestion #3 (tasks/task-2/progress.md frontmatter `status: pending`) is fixed by commit 16fe2eb (now `status: done`, consistent with the body outcome and root progress.md). No defect introduced.
- Delta 0e0a076 is artifact-only (tasks/task-2/feedback.md re-pass appending a fresh approved pass for Head 16fe2eb) and introduces no code change.
- Focused verification this pass: `go mod verify` → all modules verified; vendored leakspok analyzer .go files byte-identical to module cache leakspok@v0.3.0 (no local vendored modifications); vendored golang.org/x/sync/singleflight identical to upstream v0.20.0; no stale leakspok v0.2.0 refs in go.mod/go.sum/vendor; only repo consumer of analyzer.CacheOptions is pkg/server/app.go:65 (new fields wired). Defaults preserve v0.2.0 behavior (singleflight gated on Cache.Enabled && SingleflightEnabled; TTLJitterPercentage <= 0 disables jitter), so existing e2e cache tests pass unchanged.
- Prior-pass suggestions #1 (.env.example listing) and #2 (REDIS_ANONYMIZER_SERVICE_V2_CACHE_TOKEN docs row) remain open as non-blocking; both predate or fall outside this change's scope.
