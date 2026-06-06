package pattern

import (
	"bytes"
	"context"
	"fmt"
	"strings"
)

// Match operator constants define the available comparison strategies
// for creating pattern matchers.
const (
	// MatchOperatorEqual matches when the input is exactly equal to the reference.
	MatchOperatorEqual = "equal"
	// MatchOperatorIgnoreCaseEqual matches when the input equals the reference ignoring case.
	MatchOperatorIgnoreCaseEqual = "ignorecaseequal"
	// MatchOperatorStartsWith matches when the input starts with the reference.
	MatchOperatorStartsWith = "startswith"
	// MatchOperatorEndsWith matches when the input ends with the reference.
	MatchOperatorEndsWith = "endswith"
)

// Not returns a Pattern that matches when the provided Pattern does not match.
// It performs logical negation on the result of the wrapped pattern.
//
// Example:
//
//	notEmpty := Not(PatternFunc(func(b []byte) bool { return len(b) == 0 }))
//	// notEmpty matches any non-empty byte slice
func Not(f Pattern) Pattern {
	return NewBasePattern(
		"Not("+f.Name()+")",
		func(ctx context.Context, s []byte) bool {
			return !f.Match(ctx, s)
		},
	)
}

// All returns a Pattern that matches only when all supplied patterns match.
// It requires every pattern in the provided set to evaluate to true.
// An empty set of patterns will always return true.
//
// Example:
//
//	containsA := PatternFunc(func(b []byte) bool { return bytes.Contains(b, []byte("A")) })
//	containsB := PatternFunc(func(b []byte) bool { return bytes.Contains(b, []byte("B")) })
//	both := All(containsA, containsB) // matches only if input contains both "A" and "B"
func All(funcs ...Pattern) Pattern {
	nameParts := make([]string, 0, len(funcs))
	for _, f := range funcs {
		nameParts = append(nameParts, f.Name())
	}

	return NewBasePattern(
		"All("+strings.Join(nameParts, ", ")+")",
		func(ctx context.Context, s []byte) bool {
			for _, f := range funcs {
				if !f.Match(ctx, s) {
					return false
				}
			}
			return len(funcs) > 0
		},
	)
}

// And returns a Pattern that matches when both provided patterns match.
// It's a convenience function for combining exactly two patterns with logical AND.
//
// Example:
//
//	hasPrefix := PatternFunc(func(b []byte) bool { return bytes.HasPrefix(b, []byte("SECRET")) })
//	hasLength := PatternFunc(func(b []byte) bool { return len(b) > 10 })
//	combined := And(hasPrefix, hasLength) // matches if starts with "SECRET" AND length > 10
func And(a, b Pattern) Pattern {
	return NewBasePattern(
		"And("+a.Name()+", "+b.Name()+")",
		func(ctx context.Context, s []byte) bool {
			if !a.Match(ctx, s) {
				return false
			}

			return b.Match(ctx, s)
		},
	)
}

// Any returns a Pattern that matches when at least one of the supplied patterns matches.
// It requires only one pattern in the provided set to evaluate to true.
// An empty set of patterns will always return false.
//
// Example:
//
//	isJSON := PatternFunc(func(b []byte) bool { return bytes.HasPrefix(b, []byte("{")) })
//	isXML := PatternFunc(func(b []byte) bool { return bytes.HasPrefix(b, []byte("<")) })
//	structured := Any(isJSON, isXML) // matches if input is JSON or XML
func Any(funcs ...Pattern) Pattern {
	nameParts := make([]string, 0, len(funcs))
	for _, f := range funcs {
		nameParts = append(nameParts, f.Name())
	}

	name := "Any(" + strings.Join(nameParts, ", ") + ")"

	f := func(ctx context.Context, s []byte) bool {
		for _, f := range funcs {
			if f.Match(ctx, s) {
				return true
			}
		}
		return false
	}

	return NewBasePattern(name, f)
}

// Or returns a Pattern that matches when either of the provided patterns matches.
// It's a convenience function for combining exactly two patterns with logical OR.
//
// Example:
//
//	isPassword := PatternFunc(func(b []byte) bool { return bytes.Contains(b, []byte("password")) })
//	isSecret := PatternFunc(func(b []byte) bool { return bytes.Contains(b, []byte("secret")) })
//	sensitive := Or(isPassword, isSecret) // matches if contains "password" OR "secret"
func Or(a, b Pattern) Pattern {
	return NewBasePattern(
		"Or("+a.Name()+", "+b.Name()+")",
		func(ctx context.Context, s []byte) bool {
			if a.Match(ctx, s) {
				return true
			}

			return b.Match(ctx, s)
		},
	)
}

// Equal returns a Pattern that matches only when the input bytes are exactly equal to ref.
// The comparison is case-sensitive.
func Equal(ref []byte) Pattern {
	return NewBasePattern(
		"Equal",
		func(_ context.Context, s []byte) bool {
			return bytes.Equal(ref, s)
		},
	)
}

// IgnoreCaseEqual returns a Pattern that matches when the input bytes equal ref ignoring case.
// The comparison uses bytes.EqualFold, which supports Unicode case folding.
func IgnoreCaseEqual(ref []byte) Pattern {
	return NewBasePattern(
		"IgnoreCaseEqual",
		func(_ context.Context, s []byte) bool {
			return bytes.EqualFold(ref, s)
		},
	)
}

// StartsWith returns a Pattern that matches when the input bytes begin with ref.
// The comparison is case-sensitive.
func StartsWith(ref []byte) Pattern {
	return NewBasePattern(
		"StartsWith",
		func(_ context.Context, s []byte) bool {
			return bytes.HasPrefix(s, ref)
		},
	)
}

// EndsWith returns a Pattern that matches when the input bytes end with ref.
// The comparison is case-sensitive.
func EndsWith(ref []byte) Pattern {
	return NewBasePattern(
		"EndsWith",
		func(_ context.Context, s []byte) bool {
			return bytes.HasSuffix(s, ref)
		},
	)
}

// AtLeastN returns a Pattern that matches when at least n of the supplied patterns match.
// This enables threshold-based matching for complex pattern combinations.
//
// Parameters:
//   - n: minimum number of patterns that must match (clamped to valid range)
//   - patterns: variadic list of Pattern implementations to evaluate
//
// Behavior:
//   - If n < 1, it will be treated as 1 (at least one pattern must match)
//   - If n > len(patterns), it will be treated as len(patterns) (all patterns must match)
//   - An empty set of patterns will always return false when n >= 1
//   - Uses short-circuit evaluation for performance optimization
//
// Example:
//
//	p1 := PatternFunc(func(b []byte) bool { return bytes.Contains(b, []byte("api")) })
//	p2 := PatternFunc(func(b []byte) bool { return bytes.Contains(b, []byte("key")) })
//	p3 := PatternFunc(func(b []byte) bool { return len(b) > 20 })
//	twoOfThree := AtLeastN(2, p1, p2, p3) // matches if at least 2 of the 3 conditions are met
func AtLeastN(n int, patterns ...Pattern) Pattern {
	if n < 1 {
		n = 1
	}
	if n > len(patterns) {
		n = len(patterns)
	}

	nameParts := make([]string, 0, len(patterns))
	for _, f := range patterns {
		nameParts = append(nameParts, f.Name())
	}

	name := fmt.Sprintf("AtLeastN(%d, %s)", n, strings.Join(nameParts, ", "))
	f := func(ctx context.Context, s []byte) bool {
		passes, fails := 0, 0

		for _, f := range patterns {
			if f.Match(ctx, s) {
				passes++
			} else {
				fails++
			}

			if len(patterns)-fails < n {
				return false
			}

			if passes >= n {
				return true
			}
		}

		return false
	}

	return NewBasePattern(name, f)
}
