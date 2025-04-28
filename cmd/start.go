package cmd

import (
	"context"
	"fmt"
	"time"

	"fastcat.org/go/gdev/instance"
	"fastcat.org/go/gdev/resource"
	"fastcat.org/go/gdev/stack"
	"github.com/spf13/cobra"
)

func StackStart(cmd *cobra.Command, _ []string) error {
	svcs := stack.AllServices()
	ctx, err := resource.NewContext(cmd.Context())
	if err != nil {
		return err
	}
	// TODO: use go-pretty/v6/progress
	fmt.Printf("Starting %d services...\n", len(svcs))
	resources := make([]resource.Resource, 0, len(svcs))
	for _, svc := range svcs {
		resources = append(resources, svc.Resources(ctx)...)
	}
	fmt.Printf("Starting %d resources ...\n", len(resources))
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

func init() {
	instance.AddCommands(&cobra.Command{
		Use:   "start",
		Short: "start the stack",
		Args:  cobra.NoArgs,
		RunE:  StackStart,
	})
}
