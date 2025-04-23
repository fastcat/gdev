package main

import (
	"fmt"
	"os"

	"fastcat.org/go/gdev/cmd"
	"fastcat.org/go/gdev/instance"
	"github.com/spf13/cobra"
)

func main() {
	if err := cmd.Main(); err != nil {
		// TODO: extract a preferred exit code from the error if we can
		os.Exit(1)
	}
}

func newCustomCmd() *cobra.Command {
	return &cobra.Command{
		Use: "custom",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("this is the custom command")
		},
	}
}

func init() {
	instance.Commands = append(instance.Commands, newCustomCmd)
}
