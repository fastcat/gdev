package cmd

import (
	"fmt"

	"fastcat.org/go/gdev/pm/client"
	"github.com/spf13/cobra"
)

func pm() *cobra.Command {
	return &cobra.Command{
		Use:  "pm",
		Args: cobra.NoArgs,
		RunE: PMStatus,
	}
}

func PMStatus(cmd *cobra.Command, args []string) error {
	c := client.NewHTTP()
	err := c.Ping(cmd.Context())
	if err != nil {
		return fmt.Errorf("pm is not running: %w", err)
	}
	fmt.Println("pm is running")
	return nil
}

func init() {
	internalCommands = append(internalCommands, pm)
}
