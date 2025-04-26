package cmd

import (
	"fastcat.org/go/gdev/instance"
	"github.com/spf13/cobra"
)

func Root() *cobra.Command {
	root := &cobra.Command{
		Use:           instance.AppName(),
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       instance.Version(),
	}
	for _, c := range internalCommands {
		root.AddCommand(c())
	}
	for _, c := range instance.Commands() {
		root.AddCommand(c())
	}
	return root
}

var internalCommands []func() *cobra.Command
