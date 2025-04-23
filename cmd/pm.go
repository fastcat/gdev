package cmd

import (
	"fmt"
	"os"

	"fastcat.org/go/gdev/pm/client"
	"fastcat.org/go/gdev/pm/server"
	"github.com/spf13/cobra"
)

func pm() *cobra.Command {
	pm := &cobra.Command{
		Use:  "pm",
		Args: cobra.NoArgs,
		RunE: PMAutoStart,
	}
	pm.AddCommand(&cobra.Command{
		Use:  "status",
		Args: cobra.NoArgs,
		RunE: PMStatus,
	})
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
	summary, err := c.Summary(cmd.Context())
	if err != nil {
		return err
	}
	// TODO: format
	fmt.Println(summary)
	return nil
}

func PMAutoStart(cmd *cobra.Command, _ []string) error {
	c := client.NewHTTP()
	err := c.Ping(cmd.Context())
	if err == nil {
		fmt.Println("pm is already running")
		return nil
	}

	path := os.Args[0]
	// remove the start arg and replace it with "daemon"
	args := os.Args[1 : len(os.Args)-1]
	args = append(args, "daemon")

	return StartDaemon(cmd.Context(), "pm", path, args, map[string]string{"FOO": "BAR"})
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
