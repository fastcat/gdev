package cmd

import "github.com/spf13/cobra"

var Root = &cobra.Command{
	// will be replaced in Main
	Use: AppName,
}
