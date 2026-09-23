---
artifact: task-progress
change: 2026-09-23-wire-leakspok-cache-config
task: 4
status: done
updated: 2026-09-23
decision: accepted
---

# Task Progress: Task 4 — Document the new cache env vars

## Attempt 1 — 2026-09-23

- Date: 2026-09-23
- Outcome: done
- Created: none
- Modified:
  - docs/content/configuration.md
- Deleted: none
- Verify:
  - `uv run zensical build` (docs/) → exit 0, "No issues found", build finished in 0.94s
- Notes: Inserted the two cache env var rows immediately after the PRIVACY_CACHE_TTL row in the cache configuration table, matching the existing row style: PRIVACY_CACHE_SINGLEFLIGHT_ENABLED (default `false` — coalesces identical concurrent cache-miss computations into one per key; ignored when the cache is disabled) and PRIVACY_CACHE_TTL_JITTER_PERCENTAGE (default `0` — fraction by which each cache TTL is randomized, e.g. `0.15` = ±15%; `0` disables jitter). No other prose or tables changed. docs/uv.lock unchanged; generated docs/site output left untracked.
