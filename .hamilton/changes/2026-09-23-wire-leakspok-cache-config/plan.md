---
artifact: plan
change: 2026-09-23-wire-leakspok-cache-config
status: approved
created: 2026-09-23
author: orchestrator
decision: accepted
route_unit: null
---

# Plan: Wire leakspok cache config (singleflight + TTL jitter)

## Overview

- Change: .hamilton/changes/2026-09-23-wire-leakspok-cache-config/
- Route unit: none (not a route-unit change)
- Goal: Bump the vendored leakspok dependency from v0.2.0 to v0.3.0, which adds singleflight coalescing of concurrent cache misses and TTL jitter to the rule matching cache, then expose the two new knobs as PRIVACY_CACHE_SINGLEFLIGHT_ENABLED and PRIVACY_CACHE_TTL_JITTER_PERCENTAGE env vars wired into analyzer.CacheOptions so operators can enable them.
- Test: `task test/unit` (go test -short -race -count=1 ./...; the testcontainers-based pkg/server tests require Docker)
- Build / typecheck: `go build ./...` and `go vet ./...`
- Context notes: leakspok v0.3.0 adds two fields to analyzer.CacheOptions: TTLJitterPercentage float64 (fraction of TTL to randomize, e.g. 0.15 = ±15%; values <= 0 disable jitter) and SingleflightEnabled bool (coalesces identical concurrent cache-miss computations; ignored when Cache.Enabled is false). Both zero-value to off, so leaving the new env vars unset preserves exact v0.2.0 behavior. The only breaking change in v0.3.0 is the NewSerialRulesRuner constructor gaining a RunnerOptions parameter; this repo never calls it (it uses analyzer.MakeByteAnalyzer at pkg/server/app.go:97), so no call sites change. Cache options are built in NewFromConfig (pkg/server/app.go:64-79) from config.EnvConfig.Privacy (pkg/config/env.go), which uses caarlos0/env/v11 with envPrefix PRIVACY_. The module is vendored (vendor/ exists and Go auto-uses it), so the dependency update must end with go mod vendor; golang.org/x/sync v0.20.0 is already an indirect dependency and vendor/ will gain its singleflight package. pkg/server/app_test.go constructs config.EnvConfig with an anonymous Privacy struct type that must stay identical to config.EnvConfig.Privacy (struct tags are part of type identity), so any new field must be mirrored there or the server tests stop compiling. Env var names follow the existing convention: snake_case of the Go field name under the PRIVACY_ prefix.
- Quality notes: Task 3's server test is a construction guard rather than a red-first test because CacheOptions wiring is unobservable from outside AnonymizerServer (byteAnalyzer is unexported and the options are not retained); this matches the existing TestNewFromConfig_ConcurrencyConfigWired convention. Accepted deliberately; the behavioral value of the flags is proven by leakspok's own v0.3.0 tests.

## Tasks

### Task 1: Update leakspok to v0.3.0 and re-vendor

- Depends on: none
- Files:
  - Created: vendor/golang.org/x/sync/singleflight/** (via go mod vendor)
  - Modified: go.mod, go.sum, vendor/modules.txt, vendor/github.com/Prosus-Cyber-Xchange/leakspok/** (wholesale replacement), vendor/golang.org/x/sync/** (adds singleflight package)
  - Deleted: none expected
- Acceptance:
  - go.mod requires github.com/Prosus-Cyber-Xchange/leakspok v0.3.0
  - vendor/modules.txt records leakspok v0.3.0 and lists vendor/golang.org/x/sync/singleflight
  - No source files outside vendor/ change; the repo compiles and its unit tests pass against the new version (proves MakeByteAnalyzer's unchanged signature is the only API surface we use)
- Steps:
  1. Run `go get github.com/Prosus-Cyber-Xchange/leakspok@v0.3.0`
  2. Run `go mod tidy`; review the diff and keep changes limited to leakspok and golang.org/x/sync (x/sync stays at v0.20.0, just promoted/consumed differently)
  3. Run `go mod vendor` to regenerate vendor/ with v0.3.0 sources and the new x/sync/singleflight package
  4. Run the verify commands; if anything fails, stop and report — do not patch vendored code
- Verify: `go mod verify` → "all modules verified"; `go build ./...` → exit 0; `go test -short -race -count=1 ./...` → all packages pass (Docker required for testcontainers-based pkg/server tests)
- Commit: `chore(deps): update leakspok to v0.3.0 and re-vendor`

### Task 2: Add cache singleflight and TTL jitter env vars to config

- Depends on: Task 1
- Files:
  - Created: none
  - Modified: pkg/config/env.go, pkg/config/env_test.go, pkg/server/app_test.go
  - Deleted: none
- Acceptance:
  - config.EnvConfig.Privacy gains CacheSingleflightEnabled bool (env CACHE_SINGLEFLIGHT_ENABLED, default false) and CacheTTLJitterPercentage float64 (env CACHE_TTL_JITTER_PERCENTAGE, default 0)
  - Tests prove explicit parsing of both vars and both defaults, following the TestLoadEnv_ReadFromReplicas pattern
  - pkg/server tests still compile: the anonymous Privacy struct type in app_test.go mirrors the two new fields with identical tags
- Steps:
  1. Write failing tests in pkg/config/env_test.go; model them on TestLoadEnv_ReadFromReplicas and TestLoadEnv_WithDefaults. For explicit values use t.Setenv and:

```go
func TestLoadEnv_SingleflightAndJitter(t *testing.T) {
	t.Setenv("PRIVACY_CACHE_SINGLEFLIGHT_ENABLED", "true")
	t.Setenv("PRIVACY_CACHE_TTL_JITTER_PERCENTAGE", "0.15")

	envConfig, err := config.LoadEnv()

	require.NoError(t, err)
	assert.True(t, envConfig.Privacy.CacheSingleflightEnabled)
	assert.InDelta(t, 0.15, envConfig.Privacy.CacheTTLJitterPercentage, 1e-9)
}

func TestLoadEnv_SingleflightAndJitterDefaults(t *testing.T) {
	os.Unsetenv("PRIVACY_CACHE_SINGLEFLIGHT_ENABLED")
	os.Unsetenv("PRIVACY_CACHE_TTL_JITTER_PERCENTAGE")
	defer func() {
		os.Unsetenv("PRIVACY_CACHE_SINGLEFLIGHT_ENABLED")
		os.Unsetenv("PRIVACY_CACHE_TTL_JITTER_PERCENTAGE")
	}()

	envConfig, err := config.LoadEnv()

	require.NoError(t, err)
	assert.False(t, envConfig.Privacy.CacheSingleflightEnabled)
	assert.Zero(t, envConfig.Privacy.CacheTTLJitterPercentage)
}
```

  2. Run `go test -count=1 ./pkg/config` → expect compile failure (fields do not exist yet) — this is the red
  3. Add to the Privacy struct in pkg/config/env.go, next to CacheTTL, keeping gofmt alignment:

```go
CacheSingleflightEnabled bool    `env:"CACHE_SINGLEFLIGHT_ENABLED" envDefault:"false"`
CacheTTLJitterPercentage float64 `env:"CACHE_TTL_JITTER_PERCENTAGE" envDefault:"0"`
```

  4. Mirror the same two fields (identical tags) into the anonymous Privacy struct type in pkg/server/app_test.go so the literal in setupRedis stays type-identical to config.EnvConfig.Privacy; the struct literal values may stay zero-valued (fields can be omitted)
  5. Run the verify commands → green
- Verify: `go test -count=1 ./pkg/config` → pass; `go test -count=1 -run '^$' ./pkg/server` → compiles without running tests, exit 0
- Commit: `feat(config): add cache singleflight and TTL jitter env vars`

### Task 3: Wire singleflight and jitter fields into analyzer cache options

- Depends on: Task 2
- Files:
  - Created: none
  - Modified: pkg/server/app.go, pkg/server/app_test.go
  - Deleted: none
- Acceptance:
  - In NewFromConfig, runnerOpts.Cache carries TTLJitterPercentage from envConfig.Privacy.CacheTTLJitterPercentage and SingleflightEnabled from envConfig.Privacy.CacheSingleflightEnabled
  - A server test constructs the app with the cache enabled and both new flags set, mirroring TestNewFromConfig_ConcurrencyConfigWired (wiring is unobservable from outside AnonymizerServer, so this is a construction guard; the flag behavior itself is leakspok's, proven upstream)
- Steps:
  1. Add the guard test to pkg/server/app_test.go (not red-first by design — see Quality notes):

```go
func TestNewFromConfig_SingleflightAndJitterWired(t *testing.T) {
	cfg := setupRedis(t)
	cfg.Privacy.CacheSingleflightEnabled = true
	cfg.Privacy.CacheTTLJitterPercentage = 0.15

	app, err := server.NewFromConfig(context.Background(), server.WithEnv(cfg))
	require.NoError(t, err)
	require.NotNil(t, app)

	h := app.Handler()
	assert.NotNil(t, h)
}
```

  2. In pkg/server/app.go inside the CacheOptions literal (after DisableInMemoryCache), add exactly:

```go
TTLJitterPercentage:  a.envConfig.Privacy.CacheTTLJitterPercentage,
SingleflightEnabled:  a.envConfig.Privacy.CacheSingleflightEnabled,
```

  3. Run the verify commands
- Verify: `go vet ./...` → clean; `go test -count=1 -run 'TestNewFromConfig' ./pkg/server` → pass (Docker required for the testcontainers Redis)
- Commit: `feat(server): wire singleflight and TTL jitter into cache options`

### Task 4: Document the new cache env vars

- Depends on: Task 2
- Files:
  - Created: none
  - Modified: docs/content/configuration.md
  - Deleted: none
- Acceptance:
  - The cache configuration table in docs/content/configuration.md gains two rows matching the existing row style, placed next to PRIVACY_CACHE_TTL: PRIVACY_CACHE_SINGLEFLIGHT_ENABLED (default false — coalesces identical concurrent cache-miss computations into one per key; ignored when the cache is disabled) and PRIVACY_CACHE_TTL_JITTER_PERCENTAGE (default 0 — fraction by which each cache TTL is randomized, e.g. 0.15 = ±15%; 0 disables jitter)
  - No other prose or tables change
- Steps:
  1. Insert the two rows into the cache configuration table in docs/content/configuration.md, immediately after the PRIVACY_CACHE_TTL row, keeping column alignment and the existing terse description style
  2. Run the verify command; if the zensical CLI lacks a build subcommand, discover it with `uv run zensical --help` and use the site build/check command instead
- Verify: `uv run zensical build` in docs/ → exits 0 with no errors
- Commit: `docs: document cache singleflight and TTL jitter env vars`

## Done when

- All four tasks done, recorded in progress.md
- `go mod verify` passes and `go mod vendor` produces no diff (go.mod, go.sum, vendor/ consistent)
- `task test/unit` passes (Docker required for testcontainers server tests); `go build ./...` and `go vet ./...` clean
- Docs build succeeds
- Defaults preserve v0.2.0 behavior with the new vars unset: singleflight off, no jitter (existing e2e cache tests pass unchanged)
- All review feedback addressed
