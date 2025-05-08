package stack

import (
	"context"
	"fmt"
	"slices"

	"fastcat.org/go/gdev/resource"
	"fastcat.org/go/gdev/service"
)

// TODO: make progress printing pluggable
func Stop(ctx context.Context, includeInfrastructure bool) error {
	rc, err := resource.NewContext(ctx)
	if err != nil {
		return err
	}
	svcs := AllServices()
	if includeInfrastructure {
		// infra starts before services, StopServices will reverse this order
		svcs = append(AllInfrastructure(), svcs...)
	}
	if err := StopServices(rc, svcs...); err != nil {
		return err
	}
	// don't stop infrastructure services
	return nil
}

func StopServices(ctx *resource.Context, svcs ...service.Service) error {
	fmt.Printf("Stopping %d services...\n", len(svcs))
	resources := make([]resource.Resource, 0, len(svcs))
	for _, svc := range svcs {
		resources = append(resources, svc.Resources(ctx)...)
	}
	// stop things in reverse order
	slices.Reverse(resources)
	for _, r := range resources {
		fmt.Printf("Stopping %s...\n", r.ID())
		if err := r.Stop(ctx); err != nil {
			return fmt.Errorf("failed to stop %s: %w", r.ID(), err)
		}
	}
	return nil
}
