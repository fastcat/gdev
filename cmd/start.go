package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"fastcat.org/go/gdev/instance"
	"fastcat.org/go/gdev/service"
	"fastcat.org/go/gdev/stack"
)

func init() {
	instance.AddCommandBuilders(func() *cobra.Command {
		instance.CheckLockedDown()
		cmd := &cobra.Command{
			Use:   "start",
			Short: "start the stack",
			Args:  cobra.NoArgs,
			RunE: func(cmd *cobra.Command, args []string) error {
				// TODO: use go-pretty/v6/progress
				return stack.Start(cmd.Context(), service.WithServiceModes(service.ConfiguredModes()))
			},
		}
		f := cmd.Flags()
		for _, fn := range startFlaggers {
			fn(f)
		}
		return cmd
	})
}

var startFlaggers []func(*pflag.FlagSet)

func AddStartFlaggers(fns ...func(*pflag.FlagSet)) {
	instance.CheckCanCustomize()
	startFlaggers = append(startFlaggers, fns...)
}
