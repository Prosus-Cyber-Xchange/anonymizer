# Resolve Code TODOs Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Resolve 6 codebase TODOs — wire metrics, rename spans, remove dead rulesLoader chain, extract logger, and wire concurrency.

**Architecture:** Sequential tasks ordered by dependency — span renames first (simplest), then logger extraction + concurrency wiring (same file), then rulesLoader removal (touches 7 files), then metric reporting (depends on clean handler.go), then TODO cleanup and verification.

**Tech Stack:** Go 1.25, leakspok, tally, slog, testify, uber-go/mock

---

### Task 1: Rename Span Operation Names (TODOs #2, #3)

**Files:**
- Modify: `internal/handler/handler.go:103,157`

- [ ] **Step 1: Rename anonymize_general_request → anonymize_json_request**

Replace the span name string and remove the TODO comment on line 103.

```go
// Before (line 103):
span, ctx := monitoring.StartSpan(ctx, "anonymize_general_request") // todo: change operation name to anonymize_json_request

// After:
span, ctx := monitoring.StartSpan(ctx, "anonymize_json_request")
```

Use the GoLand/VS Code find-and-replace or a direct edit. The exact old string to match is:

```
"anonymize_general_request" // todo: change operation name to anonymize_json_request
```

Replace with:

```
"anonymize_json_request"
```

- [ ] **Step 2: Rename anonymize_text_plain → anonymize_textplain_request**

Replace the span name string and remove the TODO comment on line 157.

```go
// Before (line 157):
span, ctx := monitoring.StartSpan(ctx, "anonymize_text_plain") // todo: change operation name to anonymize_textplain_request

// After:
span, ctx := monitoring.StartSpan(ctx, "anonymize_textplain_request")
```

The exact old string to match is:

```
"anonymize_text_plain" // todo: change operation name to anonymize_textplain_request
```

Replace with:

```
"anonymize_textplain_request"
```

- [ ] **Step 3: Verify no other references to old span names exist**

Run: `rg "anonymize_general_request|anonymize_text_plain" --include='*.go'`
Expected: 0 matches

- [ ] **Step 4: Run tests to verify nothing broke**

Run: `task test/unit`
Expected: all tests pass (these are string constants, no functional change)

- [ ] **Step 5: Commit**

```bash
git add internal/handler/handler.go
git commit -m "chore: rename span operations to anonymize_json_request and anonymize_textplain_request"
```

---

### Task 2: Extract newLogger Function (TODO #5)

**Files:**
- Modify: `pkg/anonymizer/app.go:56-69`

- [ ] **Step 1: Add the newLogger function**

Add this private function to `pkg/anonymizer/app.go` — place it after `NewFromConfig` (after the closing brace of `NewFromConfig` on line 120) and before `Handler`:

```go
// newLogger creates a structured JSON logger from the environment config.
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

- [ ] **Step 2: Replace the inline logger creation block**

Replace the block at lines 55-69:

```go
// Before (lines 55-69):
	// Create default logger if not provided
	// todo: create newLogger function
	if a.logger == nil {
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
		a.logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
	}

// After:
	// Create default logger if not provided
	if a.logger == nil {
		a.logger = newLogger(envConfig)
	}
```

The exact old string to match:

```
	// Create default logger if not provided
	// todo: create newLogger function
	if a.logger == nil {
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
		a.logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
	}
```

Replace with:

```
	// Create default logger if not provided
	if a.logger == nil {
		a.logger = newLogger(envConfig)
	}
```

- [ ] **Step 3: Run tests to verify behavior is identical**

Run: `task test/unit`
Expected: all tests pass

- [ ] **Step 4: Commit**

```bash
git add pkg/anonymizer/app.go
git commit -m "refactor: extract newLogger function from NewFromConfig"
```

---

### Task 3: Wire Concurrency Options (TODO #6)

**Files:**
- Modify: `pkg/anonymizer/app.go:99-109`

- [ ] **Step 1: Uncomment and fix the concurrency wiring block**

Replace the commented-out block and the stale comment above it (lines 77 and 99-107):

```go
// Before (lines 77 and 99-107):
// Build RunnerOptions with Cache (Concurrency wiring pending leakspok.ConcurrencyOptions support).
...
		// todo: Wire concurrency options
		// runnerOpts.Concurrency = analyzer.ConcurrencyOptions{
		//     Enabled:                 envConfig.Privacy.ConcurrencyEnabled,
		//     TokenProcessing:         envConfig.Privacy.ConcurrencyTokenProcessing,
		//     RuleProcessing:          envConfig.Privacy.ConcurrencyRuleProcessing,
		//     RuleRunnerPoolSize:      envConfig.Privacy.ConcurrencyRuleRunnerPoolSize,
		//     TokenPoolSize:           envConfig.Privacy.ConcurrencyTokenPoolSize,
		//     MaxGoroutineIdleTimeout: envConfig.Privacy.ConcurrencyMaxGoroutineIdleTimeout,
		// }

// After:
// Build RunnerOptions with Cache and Concurrency.
...
	runnerOpts.Concurrency = analyzer.ConcurrencyOptions{
		Enabled:                   envConfig.Privacy.ConcurrencyEnabled,
		ConcurrentTokenProcessing: envConfig.Privacy.ConcurrencyTokenProcessing,
		ConcurrentRuleProcessing:  envConfig.Privacy.ConcurrencyRuleProcessing,
		RuleRunnerPoolSize:        envConfig.Privacy.ConcurrencyRuleRunnerPoolSize,
		TokenPoolSize:             envConfig.Privacy.ConcurrencyTokenPoolSize,
		MaxGoroutineIdleTimeout:   envConfig.Privacy.ConcurrencyMaxGoroutineIdleTimeout,
	}
```

The exact old string to match for the comment line:

```
		// Build RunnerOptions with Cache (Concurrency wiring pending leakspok.ConcurrencyOptions support).
```

Replace with:

```
		// Build RunnerOptions with Cache and Concurrency.
```

The exact old string to match for the commented block:

```
		// todo: Wire concurrency options
		// runnerOpts.Concurrency = analyzer.ConcurrencyOptions{
		//     Enabled:                 envConfig.Privacy.ConcurrencyEnabled,
		//     TokenProcessing:         envConfig.Privacy.ConcurrencyTokenProcessing,
		//     RuleProcessing:          envConfig.Privacy.ConcurrencyRuleProcessing,
		//     RuleRunnerPoolSize:      envConfig.Privacy.ConcurrencyRuleRunnerPoolSize,
		//     TokenPoolSize:           envConfig.Privacy.ConcurrencyTokenPoolSize,
		//     MaxGoroutineIdleTimeout: envConfig.Privacy.ConcurrencyMaxGoroutineIdleTimeout,
		// }
```

Replace with:

```
	runnerOpts.Concurrency = analyzer.ConcurrencyOptions{
		Enabled:                   envConfig.Privacy.ConcurrencyEnabled,
		ConcurrentTokenProcessing: envConfig.Privacy.ConcurrencyTokenProcessing,
		ConcurrentRuleProcessing:  envConfig.Privacy.ConcurrencyRuleProcessing,
		RuleRunnerPoolSize:        envConfig.Privacy.ConcurrencyRuleRunnerPoolSize,
		TokenPoolSize:             envConfig.Privacy.ConcurrencyTokenPoolSize,
		MaxGoroutineIdleTimeout:   envConfig.Privacy.ConcurrencyMaxGoroutineIdleTimeout,
	}
```

- [ ] **Step 2: Run tests to verify compilation and behavior**

Run: `task test/unit`
Expected: all tests pass, including `TestNewFromConfig_ConcurrencyConfigWired`

- [ ] **Step 3: Commit**

```bash
git add pkg/anonymizer/app.go
git commit -m "feat: wire concurrency options into leakspok RunnerOptions"
```

---

### Task 4: Remove rulesLoader Chain (TODO #4)

**Files:**
- Modify: `pkg/privacy/service.go` — remove interface, field, param, 3 methods, 2 imports
- Modify: `pkg/anonymizer/app.go:117` — drop nil argument from NewService call
- Delete: `pkg/privacy/mocks/mock_rules_loader.go`
- Modify: `pkg/privacy/service_test.go` — remove mockLoader usage, delete LoadRuleSet-dependent tests
- Modify: `internal/handler/handler_test.go` — remove mockLoader usage
- Modify: `internal/handler/handler_text_test.go` — remove mockLoader usage
- Modify: `internal/handler/handler_metrics_test.go` — remove mockLoader usage

- [ ] **Step 1: Remove rulesLoader from privacy.Service**

Edit `pkg/privacy/service.go`:

**1a:** Remove the `go:generate` directive and `PrivacyRulesLoader` interface (lines 15-21):

The exact old string to match:
```
//go:generate go run -mod=mod go.uber.org/mock/mockgen -destination=mocks/mock_rules_loader.go -package=privacymock -source=$GOFILE

// PrivacyRulesLoader defines the interface for loading privacy rules
// This interface is declared here (where it's used) rather than in the config package
type PrivacyRulesLoader interface {
	Load(ctx context.Context, serviceName string) ([]analyzer.Rule, error)
}

```
Replace with: nothing (delete these lines entirely).

**1b:** Remove the `rulesLoader` field from `Service` struct (line 25):

Old:
```go
type Service struct {
	rulesLoader  PrivacyRulesLoader
	logger       *slog.Logger
	byteAnalyzer analyzer.ByteAnalyzer
}
```
New:
```go
type Service struct {
	logger       *slog.Logger
	byteAnalyzer analyzer.ByteAnalyzer
}
```

**1c:** Update `NewService` signature and remove TODO comment:

Old:
```go
// NewService creates a new privacy service
// todo: rules loader is not used anymore, remove it
func NewService(byteAnalyzer analyzer.ByteAnalyzer, rulesLoader PrivacyRulesLoader, logger *slog.Logger) *Service {
	return &Service{
		rulesLoader:  rulesLoader,
		logger:       logger,
		byteAnalyzer: byteAnalyzer,
	}
}
```
New:
```go
// NewService creates a new privacy service
func NewService(byteAnalyzer analyzer.ByteAnalyzer, logger *slog.Logger) *Service {
	return &Service{
		logger:       logger,
		byteAnalyzer: byteAnalyzer,
	}
}
```

**1d:** Remove the `LoadRuleSet` method (lines 40-57):

The exact old string to match:
```
func (s *Service) LoadRuleSet(ctx context.Context, serviceName string, requestedEntities []string) ([]analyzer.Rule, error) {
	// Create span for loading rules
	span, ctx := monitoring.StartSpan(ctx, "rules_loader.load")
	monitoring.SetTag(span, "service.name", serviceName)
	defer span.Finish()

	allRules, err := s.rulesLoader.Load(ctx, serviceName)
	if err != nil {
		wrappedErr := fmt.Errorf("failed to load privacy rules for service %s: %w", serviceName, err)
		monitoring.SetError(span, wrappedErr)
		return nil, wrappedErr
	}

	// Filter rules based on requested entities
	rules := filterRules(allRules, requestedEntities)

	return rules, nil
}

```
Replace with: nothing (delete these lines entirely).

**1e:** Remove the deprecated `Anonymize` method (lines 91-108):

The exact old string to match:
```
// Anonymize loads privacy rules for a service and performs anonymization using an internal buffer
// Deprecated: For new code, use LoadRuleSet + AnonymizeWithRules directly with an io.Writer.
// This method creates an internal buffer and discards the anonymized output - use AnonymizeWithRules for proper output handling.
func (s *Service) Anonymize(ctx context.Context, serviceName *string, input []byte) (AnonymizeOutput, error) {
	if serviceName == nil || *serviceName == "" {
		return AnonymizeOutput{}, fmt.Errorf("service name is required")
	}

	// Load rules for the service (no entity filtering at this level)
	rules, err := s.LoadRuleSet(ctx, *serviceName, nil)
	if err != nil {
		return AnonymizeOutput{}, err
	}

	// Use a buffer for output (note: output is discarded by caller if not extracted)
	buffer := bytes.NewBuffer(nil)
	return s.AnonymizeWithRules(ctx, rules, input, buffer)
}

```
Replace with: nothing (delete these lines entirely).

**1f:** Remove the `filterRules` function (lines 110-134):

The exact old string to match:
```
// filterRules filters the rules to include only those matching the requested entities
// If requestedEntities is empty, all rules are included
func filterRules(allRules []analyzer.Rule, requestedEntities []string) []analyzer.Rule {
	// If no specific entities requested, return all rules
	if len(requestedEntities) == 0 {
		return allRules
	}

	// Create a set of requested entities (normalized to lowercase)
	requestedSet := make(map[string]struct{})
	for _, entity := range requestedEntities {
		requestedSet[strings.ToLower(entity)] = struct{}{}
	}

	filtered := make([]analyzer.Rule, 0, len(allRules))
	for _, rule := range allRules {
		ruleEntity := rule.Matcher.Entity()
		if _, found := requestedSet[strings.ToLower(string(ruleEntity))]; found {
			filtered = append(filtered, rule)
			continue
		}
	}

	return filtered
}
```
Replace with: nothing (delete these lines entirely).

**1g:** Clean up unused imports. Remove `"bytes"` and `"strings"` from the import block since they were only used by the removed code.

Old import block:
```go
import (
	"bytes"
	"context"
	"fmt"
	"github.com/Prosus-Cyber-Xchange/leakspok/analyzer"
	"io"
	"log/slog"
	"strings"

	"anonymizer-service-v2/internal/monitoring"
)
```
New import block:
```go
import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/Prosus-Cyber-Xchange/leakspok/analyzer"

	"anonymizer-service-v2/internal/monitoring"
)
```

- [ ] **Step 2: Update app.go to drop the nil argument**

In `pkg/anonymizer/app.go`, change the `NewService` call on line 117:

Old:
```go
a.privacyService = privacy.NewService(*a.byteAnalyzer, nil, a.logger)
```
New:
```go
a.privacyService = privacy.NewService(*a.byteAnalyzer, a.logger)
```

- [ ] **Step 3: Delete the generated mock file**

Run: `rm pkg/privacy/mocks/mock_rules_loader.go`

- [ ] **Step 4: Update pkg/privacy/service_test.go**

The entire test file currently depends on `LoadRuleSet` and `mockLoader`. After removing `LoadRuleSet`, `filterRules`, and the deprecated `Anonymize()` method, all existing tests become invalid (they test removed functionality). Replace the file with tests that validate `AnonymizeWithRules` directly.

Replace the entire file content:

```go
package privacy_test

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"testing"

	"anonymizer-service-v2/pkg/privacy"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/Prosus-Cyber-Xchange/leakspok/analyzer"
	"github.com/Prosus-Cyber-Xchange/leakspok/pattern"
)

func newTestByteAnalyzer(t *testing.T, logger *slog.Logger) analyzer.ByteAnalyzer {
	t.Helper()
	a, err := analyzer.MakeByteAnalyzer(context.Background(), logger, analyzer.RunnerOptions{})
	require.NoError(t, err)
	return a
}

func TestService_AnonymizeWithRules_EmptyInput(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	byteAnalyzer := newTestByteAnalyzer(t, logger)
	service := privacy.NewService(byteAnalyzer, logger)

	buffer := bytes.NewBuffer(nil)
	output, err := service.AnonymizeWithRules(context.Background(), nil, []byte{}, buffer)
	require.NoError(t, err)
	assert.Zero(t, output.Details.HasFindings)
}

func TestService_AnonymizeWithRules_NoRules(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	byteAnalyzer := newTestByteAnalyzer(t, logger)
	service := privacy.NewService(byteAnalyzer, logger)

	body := []byte("test body")
	buffer := bytes.NewBuffer(nil)
	output, err := service.AnonymizeWithRules(context.Background(), []analyzer.Rule{}, body, buffer)
	require.NoError(t, err)
	assert.Zero(t, output.Details.HasFindings)
	assert.Equal(t, body, buffer.Bytes())
}

func TestService_AnonymizeWithRules_WithEmailAnonymization(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	byteAnalyzer := newTestByteAnalyzer(t, logger)
	service := privacy.NewService(byteAnalyzer, logger)

	emailRule := analyzer.Rule{
		Name:        "email",
		Description: "Email addresses",
		Matcher:     pattern.EmailMatcher(),
		Settings: analyzer.RuleSettings{
			Strategy: analyzer.REDACT,
			Redact: &analyzer.RedactSettings{
				Placeholder: "<EMAIL_REDACTED>",
			},
		},
	}

	body := []byte("Contact us at john@example.com for support")
	buffer := bytes.NewBuffer(nil)

	output, err := service.AnonymizeWithRules(context.Background(), []analyzer.Rule{emailRule}, body, buffer)
	require.NoError(t, err)
	assert.True(t, output.Details.HasFindings)
	assert.Contains(t, string(buffer.Bytes()), "<EMAIL_REDACTED>")
	assert.NotContains(t, string(buffer.Bytes()), "john@example.com")
}

func TestService_AnonymizeWithRules_NoPIIDetected(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	byteAnalyzer := newTestByteAnalyzer(t, logger)
	service := privacy.NewService(byteAnalyzer, logger)

	emailRule := analyzer.Rule{
		Name:        "email",
		Description: "Email addresses",
		Matcher:     pattern.EmailMatcher(),
		Settings: analyzer.RuleSettings{
			Strategy: analyzer.REDACT,
			Redact: &analyzer.RedactSettings{
				Placeholder: "<EMAIL_REDACTED>",
			},
		},
	}

	body := []byte("No sensitive data here")
	buffer := bytes.NewBuffer(nil)

	output, err := service.AnonymizeWithRules(context.Background(), []analyzer.Rule{emailRule}, body, buffer)
	require.NoError(t, err)
	assert.False(t, output.Details.HasFindings)
	assert.Equal(t, body, buffer.Bytes())
}
```

- [ ] **Step 5: Update internal/handler/handler_test.go**

In each test function, remove the `mockLoader` definition and the `mockLoader` argument from `NewService`. The `privacymock` import will become unused and must be removed. `gomock` and `NewController` may also become unused — check after edits.

**5a:** Remove `privacymock` import (and `gomock` if no longer needed):

The handler_test.go file still uses `gomock` — no, actually, after removing the mockLoader, there are no gomock usages left in handler_test.go. Let me verify the existing imports:
```go
import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"anonymizer-service-v2/internal/handler"
	"anonymizer-service-v2/pkg/privacy"
	privacymock "anonymizer-service-v2/pkg/privacy/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/Prosus-Cyber-Xchange/leakspok/analyzer"
	"go.uber.org/mock/gomock"
)
```

After removing mockLoader from every test, `privacymock` and `gomock` imports are no longer needed.

Remove these import lines:
```go
	privacymock "anonymizer-service-v2/pkg/privacy/mocks"
```
```go
	"go.uber.org/mock/gomock"
```

**5b:** In each test function, remove the `ctrl` + `mockLoader` setup lines and change `NewService` calls. Here's every function affected:

For every test function, find and remove these 3 lines:
```go
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLoader := privacymock.NewMockPrivacyRulesLoader(ctrl)
```

And change each `privacy.NewService(byteAnalyzer, mockLoader, logger)` to:
```go
	privacyService := privacy.NewService(byteAnalyzer, logger)
```

The functions to update (one `NewService` call per function), plus one case where the `NewService` call is directly inside `NewHandlerWithMetrics`:

- `TestHandler_AnonymizeBatch_Success` — remove ctrl/mockLoader lines, change NewService
- `TestHandler_AnonymizeBatch_MultipleItems` — remove ctrl/mockLoader lines, change NewService
- `TestHandler_AnonymizeBatch_EmptyBatch` — remove ctrl/mockLoader lines, change NewService
- `TestHandler_AnonymizeBatch_ExceedsMaxSize` — remove ctrl/mockLoader lines, change NewService
- `TestHandler_AnonymizeBatch_InvalidJSON` — remove ctrl/mockLoader lines, change NewService
- `TestHandler_AnonymizeBatch_InvalidSettings` — remove ctrl/mockLoader lines, change NewService
- `TestHandler_AnonymizeBatch_NoPIIDetected` — remove ctrl/mockLoader lines, change NewService

- [ ] **Step 6: Update internal/handler/handler_text_test.go**

Remove `privacymock` import and `gomock` import, and remove ctrl/mockLoader from every test function.

Remove these import lines:
```go
	privacymock "anonymizer-service-v2/pkg/privacy/mocks"
```
```go
	"go.uber.org/mock/gomock"
```

In each test function, remove:
```go
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLoader := privacymock.NewMockPrivacyRulesLoader(ctrl)
```

And change `privacy.NewService(byteAnalyzer, mockLoader, logger)` to `privacy.NewService(byteAnalyzer, logger)`.

Functions affected:
- `TestHandler_Anonymize_TextPlain_Success`
- `TestHandler_Anonymize_TextPlain_MissingEntities_NoContextRules`
- `TestHandler_Anonymize_UnsupportedContentType`
- `TestHandler_Anonymize_NoContentType_DefaultsJSON`
- `TestHandler_AnonymizeBatch_NonJSON_Returns415`
- `TestHandler_Anonymize_TextPlain_WithContextRules`

- [ ] **Step 7: Update internal/handler/handler_metrics_test.go**

Remove `privacymock` import and `gomock` import. Remove ctrl/mockLoader from both test cases.

Remove these import lines:
```go
	privacymock "anonymizer-service-v2/pkg/privacy/mocks"
```
```go
	"go.uber.org/mock/gomock"
```

In the first `t.Run` block, remove:
```go
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockLoader := privacymock.NewMockPrivacyRulesLoader(ctrl)
```

And change:
```go
		privacyService := privacy.NewService(byteAnalyzer, mockLoader, logger)
```
To:
```go
		privacyService := privacy.NewService(byteAnalyzer, logger)
```

In the second `t.Run` block, remove:
```go
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockLoader := privacymock.NewMockPrivacyRulesLoader(ctrl)
```

And change:
```go
		privacyService := privacy.NewService(byteAnalyzer, mockLoader, logger)
```
To:
```go
		privacyService := privacy.NewService(byteAnalyzer, logger)
```

- [ ] **Step 8: Run tests to verify everything compiles and passes**

Run: `task test/unit`
Expected: all tests pass. If any test fails, check for:
- Missing import removals (unused `privacymock` or `gomock`)
- Lingering references to `mockLoader`
- Test functions that still call `LoadRuleSet` (these should have been replaced in service_test.go)

- [ ] **Step 9: Commit**

```bash
git add pkg/privacy/service.go pkg/anonymizer/app.go pkg/privacy/service_test.go internal/handler/handler_test.go internal/handler/handler_text_test.go internal/handler/handler_metrics_test.go
git rm pkg/privacy/mocks/mock_rules_loader.go
git commit -m "refactor: remove unused rulesLoader chain from privacy.Service"
```

---

### Task 5: Wire Metric Reporting (TODO #1)

**Files:**
- Modify: `internal/handler/handler.go` — add metric calls in 3 handler methods + add `"time"` import

- [ ] **Step 1: Remove the TODO comment on the metrics field**

Replace line 25:
```go
// Before:
metrics        PrivacyMetrics // todo: why metrics is not being used anywhere? Fix it and report metrics in the correct places

// After:
metrics        PrivacyMetrics
```

- [ ] **Step 2: Add `"time"` to the import block**

Add `"time"` to the imports. The import block currently has:

```go
import (
	...
	"log/slog"
	"net/http"
	"sort"
	"strings"
)
```

Insert `"time"` before `"log/slog"` (alphabetical with stdlib), or at the end of the stdlib group. After:

```go
import (
	...
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"time"
)
```

- [ ] **Step 3: Add metric calls in anonymizeJSON**

In the `anonymizeJSON` method, wrap the `AnonymizeWithRules` call with timing and add entity counts after success.

Find the block (around lines 132-138):
```go
output, err := h.privacyService.AnonymizeWithRules(ctx, ruleSet, req.Text, responseBuffer)
if err != nil {
	h.logger.ErrorContext(ctx, "Failed to anonymize content", slog.String("error", err.Error()))
	monitoring.SetError(span, err)
	respondError(w, http.StatusInternalServerError, "ANONYMIZATION_FAILED", err.Error())
	return
}
```

Replace with:
```go
start := time.Now()
output, err := h.privacyService.AnonymizeWithRules(ctx, ruleSet, req.Text, responseBuffer)
if err != nil {
	h.logger.ErrorContext(ctx, "Failed to anonymize content", slog.String("error", err.Error()))
	monitoring.SetError(span, err)
	respondError(w, http.StatusInternalServerError, "ANONYMIZATION_FAILED", err.Error())
	return
}
h.metrics.ObserveAnonymizationDuration(time.Since(start))
for _, entity := range output.Details.AnonymizedEntities {
	h.metrics.CountAnonymizedEntity(string(entity))
}
```

- [ ] **Step 4: Add metric calls in anonymizeTextPlain**

In the `anonymizeTextPlain` method, same pattern. Find the block (around lines 209-214):

```go
output, err := h.privacyService.AnonymizeWithRules(ctx, rules, body, responseBuffer)
if err != nil {
	monitoring.SetError(span, err)
	respondError(w, http.StatusInternalServerError, "ANONYMIZATION_FAILED", err.Error())
	return
}
```

Replace with:
```go
start := time.Now()
output, err := h.privacyService.AnonymizeWithRules(ctx, rules, body, responseBuffer)
if err != nil {
	monitoring.SetError(span, err)
	respondError(w, http.StatusInternalServerError, "ANONYMIZATION_FAILED", err.Error())
	return
}
h.metrics.ObserveAnonymizationDuration(time.Since(start))
for _, entity := range output.Details.AnonymizedEntities {
	h.metrics.CountAnonymizedEntity(string(entity))
}
```

- [ ] **Step 5: Add metric calls in AnonymizeBatch**

In the `AnonymizeBatch` method, wrap the per-item `AnonymizeWithRules` call. Find the block (around lines 294-304):

```go
responseBuffer := h.bufferPool.GetResponseBuffer()
output, err := h.privacyService.AnonymizeWithRules(ctx, ruleSet, req.Text, responseBuffer)
if err != nil {
	h.bufferPool.PutResponseBuffer(responseBuffer)
	h.logger.ErrorContext(ctx, "Failed to anonymize batch item",
		slog.Int("index", i),
		slog.String("error", err.Error()))
	monitoring.SetError(span, err)
	respondError(w, http.StatusInternalServerError, "ANONYMIZATION_FAILED", err.Error())
	return
}
```

Replace with:
```go
responseBuffer := h.bufferPool.GetResponseBuffer()
start := time.Now()
output, err := h.privacyService.AnonymizeWithRules(ctx, ruleSet, req.Text, responseBuffer)
if err != nil {
	h.bufferPool.PutResponseBuffer(responseBuffer)
	h.logger.ErrorContext(ctx, "Failed to anonymize batch item",
		slog.Int("index", i),
		slog.String("error", err.Error()))
	monitoring.SetError(span, err)
	respondError(w, http.StatusInternalServerError, "ANONYMIZATION_FAILED", err.Error())
	return
}
h.metrics.ObserveAnonymizationDuration(time.Since(start))
for _, entity := range output.Details.AnonymizedEntities {
	h.metrics.CountAnonymizedEntity(string(entity))
}
```

- [ ] **Step 6: Run unit tests**

Run: `task test/unit`
Expected: all tests pass

- [ ] **Step 7: Commit**

```bash
git add internal/handler/handler.go
git commit -m "feat: report anonymization duration and entity count metrics"
```

---

### Task 6: RunnerOptions Comment Cleanup

**Files:**
- Modify: `pkg/anonymizer/app.go:77`

- [ ] **Step 1: Update the RunnerOptions comment**

The comment at line 77 was already updated in Task 3 to say "Build RunnerOptions with Cache and Concurrency." If not already done, ensure it reads:

```go
// Build RunnerOptions with Cache and Concurrency.
```

- [ ] **Step 2: Commit**

```bash
git add pkg/anonymizer/app.go
git commit -m "chore: update RunnerOptions comment to reflect wired concurrency"
```

---

### Task 7: Final Verification

- [ ] **Step 1: Run all unit tests**

Run: `task test/unit`
Expected: PASS

- [ ] **Step 2: Run all tests (including e2e)**

Run: `task test`
Expected: all tests pass

- [ ] **Step 3: Run lint**

Run: `task lint`
Expected: no lint violations

- [ ] **Step 4: Run format**

Run: `task format`
Expected: no formatting changes needed (if changes appear, commit them)

- [ ] **Step 5: Run build**

Run: `task build`
Expected: binary compiles successfully

- [ ] **Step 6: Commit any remaining changes**

If `task format` produced changes:
```bash
git add -A
git commit -m "chore: apply gofmt formatting"
```
