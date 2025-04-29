package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"fastcat.org/go/gdev/instance"
	"fastcat.org/go/gdev/resource"
	"fastcat.org/go/gdev/stack"
)

func StackStop(cmd *cobra.Command, _ []string) error {
	svcs := stack.AllServices()
	ctx, err := resource.NewContext(cmd.Context())
	if err != nil {
		return err
	}
	// TODO: use go-pretty/v6/progress
	fmt.Printf("Stopping %d services...\n", len(svcs))
	resources := make([]resource.Resource, 0, len(svcs))
	for _, svc := range svcs {
		resources = append(resources, svc.Resources(ctx)...)
	}
	fmt.Printf("Stopping %d resources ...\n", len(resources))
	for _, r := range resources {
		fmt.Printf("Stopping %s...\n", r.ID())
		if err := r.Stop(ctx); err != nil {
			return fmt.Errorf("failed to start %s: %w", r.ID(), err)
		}
	}
	// TODO: mechanism for full stop wait?
	return nil
}

func init() {
	instance.AddCommands(&cobra.Command{
		Use:   "stop",
		Short: "stop the stack",
		Args:  cobra.NoArgs,
		RunE:  StackStop,
	})
}
