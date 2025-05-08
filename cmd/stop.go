package cmd

import (
	"context"

	"github.com/spf13/cobra"

	"fastcat.org/go/gdev/instance"
	"fastcat.org/go/gdev/stack"
)

type StackStopOptions struct {
	IncludeInfrastructure bool
}

func StackStop(ctx context.Context, opts StackStopOptions) error {
	// TODO: use go-pretty/v6/progress
	return stack.Stop(ctx, opts.IncludeInfrastructure)
	// TODO: mechanism for full stop wait?
}

func init() {
	var opts StackStopOptions
	cmd := &cobra.Command{
		Use:   "stop",
		Short: "stop the stack",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return StackStop(cmd.Context(), opts)
		},
	}
	cmd.Flags().BoolVar(&opts.IncludeInfrastructure, "include-infrastructure", opts.IncludeInfrastructure,
		"stop infrastructure too, not just normal services")
	instance.AddCommands(cmd)
}
