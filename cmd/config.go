package cmd

import (
	"github.com/spf13/cobra"

	"fastcat.org/go/gdev/instance"
)

func Config() *cobra.Command {
	cfg := &cobra.Command{
		Use: "config",
		// just a parent for other commands
	}

	for _, fn := range cfgCmdBuilders {
		cfg.AddCommand(fn())
	}

	return cfg
}

var cfgCmdBuilders []func() *cobra.Command

func AddConfigCommandBuilder(fns ...func() *cobra.Command) {
	instance.CheckCanCustomize()
	cfgCmdBuilders = append(cfgCmdBuilders, fns...)
}

func init() {
	instance.AddCommandBuilders(Config)
}
