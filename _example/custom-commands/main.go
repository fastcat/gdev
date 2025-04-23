package main

import (
	"fmt"

	"fastcat.org/go/gdev/cmd"
	"fastcat.org/go/gdev/instance"
	"github.com/spf13/cobra"
)

func main() {
	cmd.Main()
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
