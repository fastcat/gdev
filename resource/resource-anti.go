package resource

import "context"

// anti wraps a resource to ensure it is always stopped. It calls the inner
// resource's Stop method on both Start and Stop.

func Anti(inner Resource) Resource {
	return &anti{inner}
}

func IsAnti(r Resource) bool {
	_, ok := r.(*anti)
	return ok
}

type anti struct {
	r Resource
}

// ID implements Resource.
func (a *anti) ID() string {
	return "anti/" + a.r.ID()
}

// Start implements Resource.
func (a *anti) Start(ctx context.Context) error {
	return a.r.Stop(ctx)
}

// Stop implements Resource.
func (a *anti) Stop(ctx context.Context) error {
	return a.r.Stop(ctx)
}

// Ready implements Resource.
func (a *anti) Ready(ctx context.Context) (bool, error) {
	// we expect errors checking status of services we stopped
	inner, _ := a.r.Ready(ctx)
	return !inner, nil
}
