package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"fastcat.org/go/gdev/cmd"
	"fastcat.org/go/gdev/instance"
)

func main() {
	// cspell:ignore edev
	instance.SetAppName("edev")
	cmd.Main()
}

func init() {
	instance.AddCommands(&cobra.Command{
		Use: "custom",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("this is the custom command")
		},
	})
}
