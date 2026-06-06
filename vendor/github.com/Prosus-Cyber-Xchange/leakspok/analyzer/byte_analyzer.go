package analyzer

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"slices"
	"sync"

	"github.com/Prosus-Cyber-Xchange/leakspok/monitoring"
	"github.com/Prosus-Cyber-Xchange/leakspok/pattern"
)

// RuleRunner defines the interface for executing rules concurrently
type RuleRunner interface {
	// Process executes the rules and returns the first matching rule
	Process(ctx context.Context, rules []Rule, data []byte) (Rule, bool)

	Stop()
}

// ByteAnalyzer provides detection and anonymization of sensitive data within byte slices.
// It tokenizes input, evaluates configured rules against each token, and applies
// anonymization strategies (redact or mask) to matching tokens.
//
// ByteAnalyzer is safe for concurrent use. When configured with a WorkerPool,
// tokens are processed in parallel; otherwise they are processed sequentially.
type ByteAnalyzer struct {
	logger     *slog.Logger
	ruleRunner RuleRunner

	// pool is the shared goroutine pool used for concurrent token dispatch.
	// Nil when ConcurrentTokenProcessing is disabled; non-nil selects the concurrent token path.
	pool WorkerPool
}

// NewByteAnalyzer creates a new ByteAnalyzer with the provided logger and RuleRunner
func NewByteAnalyzer(logger *slog.Logger, ruleRunner RuleRunner) ByteAnalyzer {
	return ByteAnalyzer{
		logger:     logger,
		ruleRunner: ruleRunner,
	}
}

// Token represents a token in the input with its start and end positions
type Token struct {
	Start   int
	End     int
	Content []byte
}

// Len returns the length of the token in bytes.
func (t Token) Len() int {
	return t.End - t.Start
}

// AnonymizationDetails contains the results of an Anonymize operation.
// It reports whether any sensitive data was found and which entity types
// were detected and anonymized.
type AnonymizationDetails struct {
	HasFindings        bool
	DetectedEntities   []pattern.Entity
	AnonymizedEntities []pattern.Entity
}

// writeToOutput writes data to the output writer and logs any errors at warn level
func (t *ByteAnalyzer) writeToOutput(ctx context.Context, output io.Writer, data []byte) {
	_, err := output.Write(data)
	if err != nil {
		t.logger.WarnContext(ctx, "Failed to write anonymized output", slog.String("error", err.Error()))
	}
}

type anonymizationAction struct {
	token    Token
	settings RuleSettings
}

// Anonymize anonymizes all matches within the provided rule and writes to the provided writer
func (t *ByteAnalyzer) Anonymize(ctx context.Context, rules []Rule, output io.Writer, data []byte) AnonymizationDetails {
	ctx = monitoring.WithLogger(ctx, t.logger)
	ctx = monitoring.WithWorkID(ctx)

	if len(data) == 0 {
		return AnonymizationDetails{}
	}

	if len(rules) == 0 {
		return AnonymizationDetails{}
	}

	var hasFindings bool
	detectedEntities := make(map[pattern.Entity]struct{})
	anonymizedEntities := make(map[pattern.Entity]struct{})

	var actions []anonymizationAction

	content := data
	if t.pool != nil {
		actions = t.anonymizeConcurrent(ctx, rules, content, detectedEntities, anonymizedEntities)
	} else {
		actions = t.anonymizeSequential(ctx, rules, content, detectedEntities, anonymizedEntities)
	}

	hasFindings = len(actions) > 0

	if hasFindings {
		t.applyActions(ctx, output, content, actions)
	} else {
		t.writeToOutput(ctx, output, data)
	}

	return AnonymizationDetails{
		HasFindings:        hasFindings,
		DetectedEntities:   entityMapToSlice(detectedEntities),
		AnonymizedEntities: entityMapToSlice(anonymizedEntities),
	}
}

// applyActions writes the anonymized output by processing each action in order.
// It writes the original content between matches and applies the configured strategy
// (REDACT or MASK) for each matched token.
func (t *ByteAnalyzer) applyActions(ctx context.Context, output io.Writer, content []byte, actions []anonymizationAction) {
	lastPos := 0

	for _, action := range actions {
		token := action.token
		settings := action.settings

		// Write the part before this token.
		t.writeToOutput(ctx, output, content[lastPos:token.Start])

		switch settings.Strategy {
		case REDACT:
			t.writeToOutput(ctx, output, []byte(settings.Redact.Placeholder))
		case MASK:
			replacement := []byte(settings.Mask.MaskingChar)
			n := settings.Mask.MaxSize
			if n <= 0 {
				n = 1
			}

			if n > token.Len() {
				n = token.Len()
			}

			t.writeToOutput(ctx, output, bytes.Repeat(replacement, n))
			t.writeToOutput(ctx, output, content[token.Start+n:token.End])
		}

		lastPos = token.End
	}

	t.writeToOutput(ctx, output, content[lastPos:])
}

// anonymizeSequential processes tokens one at a time in iteration order.
func (t *ByteAnalyzer) anonymizeSequential(
	ctx context.Context,
	rules []Rule,
	content []byte,
	detectedEntities map[pattern.Entity]struct{},
	anonymizedEntities map[pattern.Entity]struct{},
) []anonymizationAction {
	var actions []anonymizationAction

	for token := range TokenIterator(content) {
		matchedRule, found := t.ruleRunner.Process(ctx, rules, token.Content)
		if found {
			actions = append(actions, anonymizationAction{token: token, settings: matchedRule.Settings})
			detectedEntities[matchedRule.Matcher.Entity()] = struct{}{}
			anonymizedEntities[matchedRule.Matcher.Entity()] = struct{}{}
		}
	}

	return actions
}

// anonymizeConcurrent processes tokens in parallel using the shared pool.
// Each token is submitted to the pool as a task. Results are accumulated under a
// mutex and sorted by token start position so that the output is byte-for-byte
// identical to the sequential path.
func (t *ByteAnalyzer) anonymizeConcurrent(
	ctx context.Context,
	rules []Rule,
	content []byte,
	detectedEntities map[pattern.Entity]struct{},
	anonymizedEntities map[pattern.Entity]struct{},
) []anonymizationAction {
	var (
		actions []anonymizationAction
		mu      sync.Mutex
		wg      sync.WaitGroup
	)

	for token := range TokenIterator(content) {
		tok := token // capture per-iteration value

		wg.Add(1)
		err := t.pool.Submit(func() {
			defer wg.Done()

			select {
			case <-ctx.Done():
				return
			default:
			}

			matchedRule, found := t.ruleRunner.Process(ctx, rules, tok.Content)
			if found {
				mu.Lock()
				actions = append(actions, anonymizationAction{
					token:    tok,
					settings: matchedRule.Settings,
				})
				detectedEntities[matchedRule.Matcher.Entity()] = struct{}{}
				anonymizedEntities[matchedRule.Matcher.Entity()] = struct{}{}
				mu.Unlock()
			}
		})
		if err != nil {
			// Pool is closed; undo the Add and stop submitting.
			wg.Done()
			break
		}
	}

	wg.Wait()

	// Sort by token start position to guarantee deterministic output order.
	slices.SortFunc(actions, func(a, b anonymizationAction) int {
		return a.token.Start - b.token.Start
	})

	return actions
}

// Stop releases the ByteAnalyzer's resources, stopping the rule runner and
// the goroutine pool if configured. Subsequent calls to Anonymize will
// process tokens sequentially if the pool has been released.
// Stop is safe to call concurrently with Anonymize.
func (t *ByteAnalyzer) Stop() {
	if t.ruleRunner != nil {
		t.ruleRunner.Stop()
	}

	if t.pool != nil {
		// Use a pre-cancelled context for non-blocking release.
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_ = t.pool.ReleaseContext(ctx)
	}
}

// isException checks if any of the rule exceptions match the given token
func isException(ctx context.Context, token []byte, exceptions []Exception) bool {
	for _, exception := range exceptions {
		if exception.Matcher.Match(ctx, token) {
			return true
		}
	}

	return false
}

// entityMapToSlice converts an entity set to a slice, used when building
// AnonymizationDetails from the detected and anonymized entity maps.
func entityMapToSlice(m map[pattern.Entity]struct{}) []pattern.Entity {
	list := make([]pattern.Entity, 0, len(m))
	for entity := range m {
		list = append(list, entity)
	}
	return list
}
