package analyzer

import (
	"context"
	"fmt"

	"github.com/Prosus-Cyber-Xchange/leakspok/pattern"
	"golang.org/x/sync/singleflight"
)

// singleflightCoalescer coalesces one computation per key across concurrent
// callers while enabled. Results are shared only while the call is in flight;
// the key is forgotten on completion so a later burst recomputes.
type singleflightCoalescer struct {
	group singleflight.Group
}

// tokenFlightDoer coalesces token-scoped computations across concurrent
// callers: one flight per token shares the evaluated results and the batched
// save error.
type tokenFlightDoer interface {
	doToken(ctx context.Context, key string, fn func() (map[pattern.Entity]bool, error)) (map[pattern.Entity]bool, error)
}

type disabledSingleflightCoalescer struct{}

func (disabledSingleflightCoalescer) doToken(_ context.Context, _ string, fn func() (map[pattern.Entity]bool, error)) (map[pattern.Entity]bool, error) {
	return fn()
}

func newSingleflightCoalescer() *singleflightCoalescer {
	return &singleflightCoalescer{}
}

// tokenFlightKey builds the token-scoped flight key: identical concurrent
// tokens share one flight, so a burst evaluates the token's missing entities
// and persists its negatives once.
func tokenFlightKey(data []byte) string {
	return string(data)
}

// doShared runs fn once per key among concurrent callers and shares its
// result. A caller whose context is cancelled while waiting returns the zero
// value with ctx.Err() without running fn. The key is forgotten on completion
// so a later burst recomputes; both flight flavors share this flow.
func doShared[T any](ctx context.Context, group *singleflight.Group, key string, fn func() (T, error)) (T, error) {
	ch := group.DoChan(key, func() (any, error) {
		defer group.Forget(key)
		value, err := fn()
		return value, err
	})

	var zero T
	select {
	case res := <-ch:
		value, ok := res.Val.(T)
		if !ok {
			return zero, fmt.Errorf("singleflight coalescer: unexpected result type %T", res.Val)
		}
		return value, res.Err
	case <-ctx.Done():
		return zero, ctx.Err()
	}
}

// doToken shares the computation result map and the batched-save error among
// concurrent callers of the same token.
// A caller whose context is cancelled while waiting returns (nil, ctx.Err())
// without running fn.
func (c *singleflightCoalescer) doToken(ctx context.Context, key string, fn func() (map[pattern.Entity]bool, error)) (map[pattern.Entity]bool, error) {
	return doShared(ctx, &c.group, key, fn)
}
