package cmd

import (
	"fastcat.org/go/gdev/instance"
	"github.com/spf13/cobra"
)

var Root = &cobra.Command{
	// will be replaced in Main
	Use: instance.AppName,
}
