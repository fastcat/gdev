package stack

import (
	"context"
	"fmt"
	"time"

	"fastcat.org/go/gdev/resource"
	"fastcat.org/go/gdev/service"
)

// Start starts the stack with the given options.
//
// Options must be of type [service.ContextOption] or [resource.ContextOption],
// and will be passed to [service.NewContext] and [resource.NewContext]
// respectively.
func Start(ctx context.Context, opts ...any) error {
	// TODO: make progress printing pluggable
	var svcOpts []service.ContextOption
	var rcOpts []resource.ContextOption
	for _, opt := range opts {
		switch o := opt.(type) {
		case service.ContextOption:
			svcOpts = append(svcOpts, o)
		case resource.ContextOption:
			rcOpts = append(rcOpts, o)
		default:
			return fmt.Errorf(
				"unexpected option type %T, expected service.ContextOption or resource.ContextOption",
				o,
			)
		}
	}
	// TODO: don't double-layer if input already has resource/service context layers
	// TODO: validate we have all the required service options
	ctx = service.NewContext(ctx, svcOpts...)
	ctx, err := resource.NewContext(ctx, rcOpts...)
	if err != nil {
		return err
	}
	infra, svcs := AllInfrastructure(), AllServices()
	if err := preStart(ctx, infra, svcs); err != nil {
		return fmt.Errorf("error running pre-start hooks: %w", err)
	}
	if err := StartServices(ctx, "infrastructure", infra...); err != nil {
		return err
	}
	if err := StartServices(ctx, "stack", svcs...); err != nil {
		return err
	}
	return nil
}

func StartServices(ctx context.Context, kind string, svcs ...service.Service) error {
	if len(svcs) == 0 {
		return nil
	}
	fmt.Printf("Starting %d services (%s)...\n", len(svcs), kind)
	resources := make([]resource.Resource, 0, len(svcs))
	for _, svc := range svcs {
		rs := svc.Resources(ctx)
		// if this service is disabled, force all its resources to be stopped
		if m, _ := service.ServiceMode(ctx, svc.Name()); m == service.ModeDisabled {
			for i, r := range rs {
				if !resource.IsAnti(r) {
					rs[i] = resource.Anti(r)
				}
			}
		}
		resources = append(resources, rs...)
	}
	for _, r := range resources {
		fmt.Printf("Starting %s...\n", r.ID())
		if err := r.Start(ctx); err != nil {
			return fmt.Errorf("failed to start %s: %w", r.ID(), err)
		}
	}
	fmt.Printf("Waiting for ready ...\n")
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	// TODO: wait for all to be ready in a single pass, instead of all being ready
	// sequentially, catches crash loops better
	for _, r := range resources {
		fmt.Printf("Waiting on %s ", r.ID())
		for {
			if ready, err := r.Ready(ctx); err != nil {
				fmt.Println(" FAILED")
				return fmt.Errorf("error checking %s for ready: %w", r.ID(), err)
			} else if ready {
				fmt.Println("OK")
				break
			}
			select {
			case <-ctx.Done():
				return context.Cause(ctx)
			case <-ticker.C:
				// retry
				fmt.Print(".")
			}
		}
	}
	return nil
}
