package cmd

import (
	"fastcat.org/go/gdev/instance"
	"github.com/spf13/cobra"
)

func Root() *cobra.Command {
	return &cobra.Command{
		Use: instance.AppName,
	}
}
