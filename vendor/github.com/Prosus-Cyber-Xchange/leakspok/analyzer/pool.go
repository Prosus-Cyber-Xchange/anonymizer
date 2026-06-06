package analyzer

import (
	"context"
	"log/slog"

	"github.com/panjf2000/ants/v2"
)

//go:generate mockgen -destination=mocks/mock_worker_pool.go -package=analyzermock github.com/Prosus-Cyber-Xchange/leakspok/analyzer WorkerPool

// WorkerPool abstracts a bounded goroutine pool.
// Implementations must be safe for concurrent use.
type WorkerPool interface {
	// Submit dispatches a task for execution by a pool worker.
	// If the pool is at capacity, Submit blocks until a worker becomes available.
	// Returns an error if the pool has been released (closed).
	Submit(task func()) error

	// ReleaseContext gracefully shuts down the pool. It waits for all in-flight
	// tasks to complete or until the provided context is cancelled/expired.
	// After ReleaseContext, subsequent Submit calls return an error.
	// ReleaseContext is idempotent.
	ReleaseContext(ctx context.Context) error
}

// AntsWorkerPool is a [WorkerPool] backed by github.com/panjf2000/ants/v2.
type AntsWorkerPool struct {
	pool *ants.Pool
}

// NewAntsWorkerPool creates a [WorkerPool] backed by ants/v2.
// size must be > 0.
func NewAntsWorkerPool(size int, logger *slog.Logger, opts ...ants.Option) (WorkerPool, error) {
	defaults := []ants.Option{
		ants.WithPanicHandler(func(err interface{}) {
			logger.Error("worker pool panic recovered", "panic", err)
		}),
	}

	p, err := ants.NewPool(size, append(defaults, opts...)...)
	if err != nil {
		return nil, err
	}

	return &AntsWorkerPool{pool: p}, nil
}

// Submit implements [WorkerPool].
func (w *AntsWorkerPool) Submit(task func()) error {
	return w.pool.Submit(task)
}

// ReleaseContext implements [WorkerPool].
func (w *AntsWorkerPool) ReleaseContext(ctx context.Context) error {
	return w.pool.ReleaseContext(ctx)
}
