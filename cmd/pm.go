package cmd

import (
	"fmt"
	"os"
	"slices"

	"fastcat.org/go/gdev/pm/client"
	"fastcat.org/go/gdev/pm/server"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
)

func pm() *cobra.Command {
	pm := &cobra.Command{
		Use:   "pm",
		Short: "starts the pm daemon if it isn't already running",
		Args:  cobra.NoArgs,
		RunE:  PMAutoStart,
	}
	pm.AddCommand(&cobra.Command{
		Use:   "status",
		Short: "show pm services",
		Args:  cobra.NoArgs,
		RunE:  PMStatus,
	})
	pm.AddCommand(&cobra.Command{
		Use:   "terminate",
		Short: "terminate pm daemon and any children",
		Args:  cobra.NoArgs,
		RunE:  PMTerminate,
	})
	pm.AddCommand(&cobra.Command{
		Use:    "daemon",
		Short:  "runs the pm daemon in the foreground",
		Args:   cobra.NoArgs,
		RunE:   pmDaemon,
		Hidden: true,
	})
	return pm
}

func PMStatus(cmd *cobra.Command, _ []string) error {
	c := client.NewHTTP()
	if err := c.Ping(cmd.Context()); err != nil {
		return fmt.Errorf("pm is not running: %w", err)
	}
	summary, err := c.Summary(cmd.Context())
	if err != nil {
		return err
	}
	tw := table.NewWriter()
	tw.SetStyle(table.StyleColoredBlueWhiteOnBlack)
	tw.SetOutputMirror(os.Stdout)
	tw.AppendHeader(table.Row{"Name", "State", "Pid", "Healthy"})
	tw.AppendSeparator()
	for _, c := range summary {
		h := "❔"
		if c.Healthy != nil {
			if *c.Healthy {
				h = "👍"
			} else {
				h = "❌"
			}
		}
		tw.AppendRow(table.Row{c.Name, c.State, c.Pid, h})
	}
	tw.Render()
	return nil
}

func PMAutoStart(cmd *cobra.Command, _ []string) error {
	c := client.NewHTTP()
	if err := c.Ping(cmd.Context()); err == nil {
		fmt.Println("pm is already running")
		return nil
	}

	path := os.Args[0]
	// find the pm arg and append daemon after it
	args := os.Args[1:]
	pmIdx := slices.Index(args, "pm")
	if pmIdx < 0 {
		panic("pm autostart invoked from bad cli args")
	}
	args = append(slices.Clip(args[:pmIdx+1]), "daemon")

	return StartDaemon(cmd.Context(), "pm", path, args, map[string]string{"FOO": "BAR"})
}

func PMTerminate(cmd *cobra.Command, _ []string) error {
	c := client.NewHTTP()
	if err := c.Ping(cmd.Context()); err != nil {
		// TODO: check the specific error better
		fmt.Println("pm no running")
		return nil
	}
	if err := c.Terminate(cmd.Context()); err != nil {
		return fmt.Errorf("failed to terminate pm daemon: %w", err)
	}
	return nil
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
