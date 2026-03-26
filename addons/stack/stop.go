package stack

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"

	"fastcat.org/go/gdev/progress"
	"fastcat.org/go/gdev/resource"
	"fastcat.org/go/gdev/service"
)

type StackStopOptions struct {
	IncludeInfrastructure bool
	Exclude               []string
	Parallel              bool
}

func StackStop(ctx context.Context, opts StackStopOptions) error {
	ctx, stop := progress.StartWriter(ctx)
	defer stop()

	ctx, err := resource.NewContext(ctx)
	if err != nil {
		return err
	}
	deleteFunc := func(s service.Service) bool {
		return slices.Contains(opts.Exclude, s.Name())
	}
	// do services & infra separately in case parallel was requested. infra after
	// services because we stop things in reverse of start order.
	svcs := slices.DeleteFunc(AllServices(), deleteFunc)
	if err := StopServices(ctx, opts, "stack", svcs...); err != nil {
		return err
	}
	if opts.IncludeInfrastructure {
		svcs := slices.DeleteFunc(AllInfrastructure(), deleteFunc)
		if err := StopServices(ctx, opts, "infrastructure", svcs...); err != nil {
			return err
		}
	}

	// TODO: mechanism for full stop wait?

	return nil
}

// Deprecated: use StackStop instead, newer options are not available here
//
//go:fix inline
func Stop(
	ctx context.Context,
	includeInfrastructure bool,
	exclude []string,
) error {
	return StackStop(ctx, StackStopOptions{
		IncludeInfrastructure: includeInfrastructure,
		Exclude:               exclude,
	})
}

func StopServices(ctx context.Context, opts StackStopOptions, kind string, svcs ...service.Service) error {
	pt := &progress.Tracker{
		Message: fmt.Sprintf("Stopping %d services (%s)...", len(svcs), kind),
		Total:   int64(len(svcs)),
		Units:   progress.UnitsDefault,
	}
	progress.AddTracker(ctx, pt)

	resources := make([]resource.Resource, 0, len(svcs))
	var errs []error
	for _, svc := range svcs {
		r, err := svc.Resources(ctx)
		if err != nil {
			errs = append(errs, err)
		}
		resources = append(resources, r...)
	}
	// stop everything we can, in reverse order, don't return errors until the end
	slices.Reverse(resources)
	// buffer 1 so we can do the non-parallel stop without extra shenanigans
	errCh := make(chan error, 1)
	var wg sync.WaitGroup
	for _, r := range resources {
		pt.UpdateMessage(fmt.Sprintf("Stopping %s", r.ID()))
		wg.Go(func() {
			if err := r.Stop(ctx); err != nil {
				pt.MarkAsErrored()
				errCh <- fmt.Errorf("failed to stop %s: %w", r.ID(), err)
			}
			pt.Increment(1)
		})
		if !opts.Parallel {
			wg.Wait()
			select {
			case err := <-errCh:
				errs = append(errs, err)
			default:
			}
		}
	}
	if opts.Parallel {
		pt.UpdateMessage(fmt.Sprintf("Waiting for %d services (%s) to stop", len(svcs), kind))
		go func() { defer close(errCh); wg.Wait() }()
		for err := range errCh {
			errs = append(errs, err)
		}
	}
	if len(errs) == 0 {
		pt.UpdateMessage(fmt.Sprintf("Stopped %d services (%s)", len(svcs), kind))
		pt.MarkAsDone()
		return nil
	}

	return errors.Join(errs...)
}
