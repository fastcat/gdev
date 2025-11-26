package cmd

import (
	"context"

	"github.com/spf13/cobra"

	"fastcat.org/go/gdev/instance"
	"fastcat.org/go/gdev/stack"
)

type StackStopOptions struct {
	IncludeInfrastructure bool
	Exclude               []string
}

func StackStop(ctx context.Context, opts StackStopOptions) error {
	// TODO: use go-pretty/v6/progress
	return stack.Stop(ctx, opts.IncludeInfrastructure, opts.Exclude)
	// TODO: mechanism for full stop wait?
}

func init() {
	var opts StackStopOptions
	scd := cobra.ShellCompDirectiveNoFileComp
	cmd := &cobra.Command{
		Use:   "stop",
		Short: "stop the stack",
		Args:  cobra.NoArgs,
		CompletionOptions: cobra.CompletionOptions{
			DefaultShellCompDirective: &scd,
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return StackStop(cmd.Context(), opts)
		},
	}
	cmd.Flags().BoolVar(&opts.IncludeInfrastructure, "include-infrastructure", opts.IncludeInfrastructure,
		"stop infrastructure too, not just normal services")
	cmd.Flags().StringSliceVar(&opts.Exclude, "exclude", opts.Exclude,
		"exclude these resources from stopping (suffix match after slash)")
	instance.AddCommands(cmd)
}
