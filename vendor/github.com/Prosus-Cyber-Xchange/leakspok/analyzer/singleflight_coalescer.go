package analyzer

import (
	"context"
	"fmt"

	"golang.org/x/sync/singleflight"
)

// singleflightCoalescer coalesces one computation per key across concurrent
// callers while enabled. Results are shared only while the call is in flight;
// the key is forgotten on completion so a later burst recomputes.
type singleflightCoalescer struct {
	group singleflight.Group
}

type singleflightDoer interface {
	do(ctx context.Context, key string, fn func() (bool, error)) (bool, error)
}

type disabledSingleflightCoalescer struct{}

func (disabledSingleflightCoalescer) do(_ context.Context, _ string, fn func() (bool, error)) (bool, error) {
	return fn()
}

func newSingleflightCoalescer() *singleflightCoalescer {
	return &singleflightCoalescer{}
}

func singleflightKey(entity string, data []byte) string {
	return entity + ":" + string(data)
}

// do runs fn once per key among concurrent callers and shares its result. A
// caller whose context is cancelled while waiting returns ctx.Err() without
// running fn.
func (c *singleflightCoalescer) do(ctx context.Context, key string, fn func() (bool, error)) (bool, error) {
	ch := c.group.DoChan(key, func() (any, error) {
		defer c.group.Forget(key)
		matched, err := fn()
		return matched, err
	})

	select {
	case res := <-ch:
		matched, ok := res.Val.(bool)
		if !ok {
			return false, fmt.Errorf("singleflight coalescer: unexpected result type %T", res.Val)
		}
		return matched, res.Err
	case <-ctx.Done():
		return false, ctx.Err()
	}
}
