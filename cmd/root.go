package cmd

import (
	"fastcat.org/go/gdev/instance"
	"github.com/spf13/cobra"
)

func Root() *cobra.Command {
	root := &cobra.Command{
		Use: instance.AppName,
	}
	for _, c := range instance.Commands {
		root.AddCommand(c())
	}
	return root
}
