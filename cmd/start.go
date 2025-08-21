package cmd

import (
	"github.com/spf13/cobra"

	"fastcat.org/go/gdev/instance"
	"fastcat.org/go/gdev/service"
	"fastcat.org/go/gdev/stack"
)

func init() {
	instance.AddCommands(&cobra.Command{
		Use:   "start",
		Short: "start the stack",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			// TODO: use go-pretty/v6/progress
			return stack.Start(cmd.Context(), service.WithServiceModes(service.ConfiguredModes()))
		},
	})
}
