// Package fixture exercises Go declaration and call normalization.
package fixture

import (
    "context"
    "example/support"
)

// WorkContract defines operations for a generic worker.
type WorkContract[T any] interface {
    run(T) T
    Close() error
}

// BaseWorker is embedded by Worker.
type BaseWorker struct{}

// Worker stores a value and implements WorkContract.
type Worker[T any] struct {
    BaseWorker
    current T
}

// run returns the item after calling helper twice.
func (w *Worker[T]) run(item T) T {
    w.helper(item)
    w.helper(item)
    return item
}

func (w *Worker[T]) helper(item T) T {
    return item
}

func (w Worker[T]) async_probe(ctx context.Context, item T) T {
    done := make(chan T, 1)
    go func(value T) {
        support.Touch(value)
        done <- value
    }(item)
    select {
    case <-ctx.Done():
        return item
    case value := <-done:
        return value
    }
}

func (w Worker[T]) nested_probe(item T) T {
    nested := func(value T) T { return value }
    return nested(item)
}

// New creates a worker with a generic value.
func New[T any](current T) *Worker[T] {
    return &Worker[T]{current: current}
}

const DefaultCapacity = 4

var DefaultContext context.Context

