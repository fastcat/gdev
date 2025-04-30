package stack

import (
	"context"
	"fmt"
	"time"

	"fastcat.org/go/gdev/resource"
	"fastcat.org/go/gdev/service"
)

// TODO: make progress printing pluggable
func Start(ctx context.Context) error {
	rc, err := resource.NewContext(ctx)
	if err != nil {
		return err
	}
	if err := StartServices(rc, AllInfrastructure()...); err != nil {
		return err
	}
	if err := StartServices(rc, AllServices()...); err != nil {
		return err
	}
	return nil
}

func StartServices(ctx *resource.Context, svcs ...service.Service) error {
	fmt.Printf("Starting %d services...\n", len(svcs))
	resources := make([]resource.Resource, 0, len(svcs))
	for _, svc := range svcs {
		resources = append(resources, svc.Resources(ctx)...)
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
