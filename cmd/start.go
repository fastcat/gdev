package cmd

import (
	"github.com/spf13/cobra"

	"fastcat.org/go/gdev/instance"
	"fastcat.org/go/gdev/stack"
)

func StackStart(cmd *cobra.Command, _ []string) error {
	// TODO: use go-pretty/v6/progress
	return stack.Start(cmd.Context())
}

func init() {
	instance.AddCommands(&cobra.Command{
		Use:   "start",
		Short: "start the stack",
		Args:  cobra.NoArgs,
		RunE:  StackStart,
	})
}
