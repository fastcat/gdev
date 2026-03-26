package stack

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/spf13/pflag"

	"fastcat.org/go/gdev/cmd"
	"fastcat.org/go/gdev/instance"
	"fastcat.org/go/gdev/progress"
	"fastcat.org/go/gdev/resource"
	"fastcat.org/go/gdev/service"
)

var startFlaggers []func(*pflag.FlagSet, cmd.FlagCompletionRegistrar) error

func AddStartFlaggers(fns ...func(*pflag.FlagSet, cmd.FlagCompletionRegistrar) error) {
	instance.CheckCanCustomize()
	startFlaggers = append(startFlaggers, fns...)
}

// Start starts the stack with the given options.
//
// Options must be of type [service.ContextOption] or [resource.ContextOption],
// and will be passed to [service.NewContext] and [resource.NewContext]
// respectively.
func Start(ctx context.Context, opts ...any) error {
	ctx, stop := progress.StartWriter(ctx)
	defer stop()

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
	infra, svcs, err := preStart(ctx)
	if err != nil {
		return fmt.Errorf("error preparing services: %w", err)
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
	pt := &progress.Tracker{
		Message: fmt.Sprintf("Starting %d services (%s)", len(svcs), kind),
		Total:   int64(len(svcs)),
		Units:   progress.UnitsDefault,
	}
	progress.AddTracker(ctx, pt)
	resources := make([]resource.Resource, 0, len(svcs))
	var errs []error
	for _, svc := range svcs {
		rs, err := svc.Resources(ctx)
		if err != nil {
			pt.MarkAsErrored()
			errs = append(errs, err)
		}
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
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	for _, r := range resources {
		pt.UpdateMessage(fmt.Sprintf("Starting %s...", r.ID()))
		if err := r.Start(ctx); err != nil {
			pt.MarkAsErrored()
			return fmt.Errorf("failed to start %s: %w", r.ID(), err)
		}
		pt.Increment(1)
	}

	if kind != "infrastructure" && !service.NoServiceWait(ctx) {
		if err := waitResources(ctx, resources); err != nil {
			return err
		}
	}

	pt.UpdateMessage(fmt.Sprintf("Started %d services (%s)", len(svcs), kind))
	pt.MarkAsDone()
	return nil
}

func waitResources(ctx context.Context, resources []resource.Resource) error {
	fmt.Printf("Waiting for ready ...\n")
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	// TODO: wait for all to be ready in a single pass, instead of all being ready
	// sequentially, catches crash loops better
	printedDots := false
	for _, r := range resources {
		fmt.Printf("Waiting on %s ", r.ID())
		for {
			if ready, err := r.Ready(ctx); err != nil {
				if printedDots {
					fmt.Print(" ")
				}
				fmt.Println("FAILED")
				return fmt.Errorf("error checking %s for ready: %w", r.ID(), err)
			} else if ready {
				if printedDots {
					fmt.Print(" ")
				}
				fmt.Println("OK")
				break
			}
			select {
			case <-ctx.Done():
				return context.Cause(ctx)
			case <-ticker.C:
				// retry
				fmt.Print(".")
				printedDots = true
			}
		}
	}
	return nil
}
