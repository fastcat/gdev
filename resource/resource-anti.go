package resource

// Anti wraps a resource to ensure it is always stopped. It calls the inner
// resource's Stop method on both Start and Stop.
type Anti struct {
	Inner Resource
}

var _ Resource = (*Anti)(nil)

// ID implements Resource.
func (a *Anti) ID() string {
	return "anti/" + a.Inner.ID()
}

// Start implements Resource.
func (a *Anti) Start(ctx *Context) error {
	return a.Inner.Stop(ctx)
}

// Stop implements Resource.
func (a *Anti) Stop(ctx *Context) error {
	return a.Inner.Stop(ctx)
}

// Ready implements Resource.
func (a *Anti) Ready(ctx *Context) (bool, error) {
	// we expect errors checking status of services we stopped
	inner, _ := a.Inner.Ready(ctx)
	return !inner, nil
}
