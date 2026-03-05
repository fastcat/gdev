package stack

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"

	"fastcat.org/go/gdev/resource"
	"fastcat.org/go/gdev/service"
)

type StackStopOptions struct {
	IncludeInfrastructure bool
	Exclude               []string
	Parallel              bool
}

func StackStop(ctx context.Context, opts StackStopOptions) error {
	// TODO: use go-pretty/v6/progress
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
	if err := StopServices(ctx, opts, svcs...); err != nil {
		return err
	}
	if opts.IncludeInfrastructure {
		svcs := slices.DeleteFunc(AllInfrastructure(), deleteFunc)
		if err := StopServices(ctx, opts, svcs...); err != nil {
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

func StopServices(ctx context.Context, opts StackStopOptions, svcs ...service.Service) error {
	fmt.Printf("Stopping %d services...\n", len(svcs))
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
		fmt.Printf("Stopping %s...\n", r.ID())
		wg.Go(func() {
			if err := r.Stop(ctx); err != nil {
				errCh <- fmt.Errorf("failed to stop %s: %w", r.ID(), err)
			}
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
		go func() { defer close(errCh); wg.Wait() }()
		for err := range errCh {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
