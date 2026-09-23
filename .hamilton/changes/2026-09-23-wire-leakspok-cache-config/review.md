---
artifact: review
change: 2026-09-23-wire-leakspok-cache-config
status: approved
updated: 2026-09-23
decision: accepted
---
Base: f2b5c5ecd36726311ddbdfe414086864db55e4f0
Head: af6552780410f29120a3a8cd85e6342ab5ee2d4a
Verdict: approved
### Blocking
- none
### Suggestions
- [.env.example] The two new cache vars (PRIVACY_CACHE_SINGLEFLIGHT_ENABLED, PRIVACY_CACHE_TTL_JITTER_PERCENTAGE) are not listed in the sample env file. Optional: add them next to PRIVACY_CACHE_TTL for operator discoverability; plan.md scoped docs rows to docs/content/configuration.md only, so this is a non-blocking convenience improvement.
- [docs/content/configuration.md:41] Pre-existing row `REDIS_ANONYMIZER_SERVICE_V2_CACHE_TOKEN` does not match the actual env name `PRIVACY_REDIS_CACHE_TOKEN` (env.go:33 RedisToken tag `REDIS_CACHE_TOKEN` under envPrefix `PRIVACY_`). Predates this branch; not part of Task 4 scope. Optional consistency fix in a follow-up.
- [tasks/task-2/progress.md:3] Frontmatter `status: pending` contradicts the body "Outcome: done" and the root progress.md done status for Task 2. Bookkeeping hygiene only; does not affect the approved task evidence.
