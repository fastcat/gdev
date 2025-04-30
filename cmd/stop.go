package cmd

import (
	"github.com/spf13/cobra"

	"fastcat.org/go/gdev/instance"
	"fastcat.org/go/gdev/stack"
)

func StackStop(cmd *cobra.Command, _ []string) error {
	// TODO: use go-pretty/v6/progress
	return stack.Stop(cmd.Context())
	// TODO: mechanism for full stop wait?
}

func init() {
	instance.AddCommands(&cobra.Command{
		Use:   "stop",
		Short: "stop the stack",
		Args:  cobra.NoArgs,
		RunE:  StackStop,
	})
}
