package stack

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"fastcat.org/go/gdev/resource"
	"fastcat.org/go/gdev/service"
)

// TODO: make progress printing pluggable
func Stop(
	ctx context.Context,
	includeInfrastructure bool,
	exclude []string,
) error {
	ctx, err := resource.NewContext(ctx)
	if err != nil {
		return err
	}
	svcs := AllServices()
	if includeInfrastructure {
		// infra starts before services, StopServices will reverse this order
		svcs = append(AllInfrastructure(), svcs...)
	}
	if len(exclude) != 0 {
		filtered := make([]service.Service, 0, len(svcs))
		for _, svc := range svcs {
			if !slices.Contains(exclude, svc.Name()) {
				filtered = append(filtered, svc)
			}
		}
		svcs = filtered
	}
	if err := StopServices(ctx, svcs...); err != nil {
		return err
	}
	// don't stop infrastructure services
	return nil
}

func StopServices(ctx context.Context, svcs ...service.Service) error {
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
	for _, r := range resources {
		fmt.Printf("Stopping %s...\n", r.ID())
		if err := r.Stop(ctx); err != nil {
			errs = append(errs, fmt.Errorf("failed to stop %s: %w", r.ID(), err))
		}
	}
	return errors.Join(errs...)
}
