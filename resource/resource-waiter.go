package resource

import (
	"context"
	"time"
)

type waitResource struct {
	name  string
	ready func(context.Context) (bool, error)
}

// NewWaitResource creates a resource that blocks during start until the
// provided ready function passes. That function will also provide the
// implementation of Ready. Stop is a no-op.
// TODO: allow customizing the poll interval
func NewWaitResource(name string, ready func(context.Context) (bool, error)) *waitResource {
	return &waitResource{
		name:  name,
		ready: ready,
	}
}

// ID implements Resource.
func (r *waitResource) ID() string {
	return "Wait/" + r.name
}

// Ready implements Resource.
func (r *waitResource) Ready(ctx context.Context) (bool, error) {
	return r.ready(ctx)
}

// Start implements Resource.
func (r *waitResource) Start(ctx context.Context) error {
	retryTicker := time.NewTicker(250 * time.Millisecond)
	defer retryTicker.Stop()
	for {
		if ready, err := r.Ready(ctx); err != nil {
			return err
		} else if ready {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-retryTicker.C:
			// continue & re-check
		}
	}
}

// Stop implements Resource.
func (r *waitResource) Stop(context.Context) error {
	return nil
}
