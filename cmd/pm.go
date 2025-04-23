package cmd

import (
	"fmt"

	"fastcat.org/go/gdev/pm/client"
	"fastcat.org/go/gdev/pm/server"
	"github.com/spf13/cobra"
)

func pm() *cobra.Command {
	pm := &cobra.Command{
		Use:  "pm",
		Args: cobra.NoArgs,
		RunE: PMStatus,
	}
	pm.AddCommand(&cobra.Command{
		Use:    "daemon",
		Args:   cobra.NoArgs,
		RunE:   pmDaemon,
		Hidden: true,
	})
	return pm
}

func PMStatus(cmd *cobra.Command, _ []string) error {
	c := client.NewHTTP()
	err := c.Ping(cmd.Context())
	if err != nil {
		return fmt.Errorf("pm is not running: %w", err)
	}
	fmt.Println("pm is running")
	return nil
}

func PMStart(cmd *cobra.Command, _ []string) error {
	c := client.NewHTTP()
	err := c.Ping(cmd.Context())
	if err == nil {
		return fmt.Errorf("pm is already running")
	}

	// TODO: invoke "argv[0] pm daemon"
	panic("unimplemented")
}

func pmDaemon(cmd *cobra.Command, _ []string) error {
	d, err := server.NewHTTP()
	if err != nil {
		return err
	}
	return d.Run(cmd.Context())
}

func init() {
	internalCommands = append(internalCommands, pm)
}
