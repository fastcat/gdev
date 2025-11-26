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
		scd := cobra.ShellCompDirectiveNoFileComp
		cmd := &cobra.Command{
			Use:   "start",
			Short: "start the stack",
			Args:  cobra.NoArgs,
			CompletionOptions: cobra.CompletionOptions{
				DefaultShellCompDirective: &scd,
			},
			RunE: func(cmd *cobra.Command, args []string) error {
				// TODO: use go-pretty/v6/progress
				return stack.Start(cmd.Context(), service.WithServiceModes(service.ConfiguredModes()))
			},
		}
		f := cmd.Flags()
		for _, fn := range startFlaggers {
			if err := fn(f, cmd.RegisterFlagCompletionFunc); err != nil {
				panic(err)
			}
		}
		return cmd
	})
}

var startFlaggers []func(*pflag.FlagSet, FlagCompletionRegistrar) error

type FlagCompletionRegistrar func(string, cobra.CompletionFunc) error

func AddStartFlaggers(fns ...func(*pflag.FlagSet, FlagCompletionRegistrar) error) {
	instance.CheckCanCustomize()
	startFlaggers = append(startFlaggers, fns...)
}
