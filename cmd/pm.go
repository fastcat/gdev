package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"

	"fastcat.org/go/gdev/pm/api"
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

	pm.AddCommand(pmAdd())
	pm.AddCommand(&cobra.Command{
		Use:   "start",
		Short: "starts one or more pm service(s)",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := client.NewHTTP()
			for _, name := range args {
				if stat, err := c.StartChild(cmd.Context(), name); err != nil {
					return fmt.Errorf("failed to start %s: %w", name, err)
				} else {
					// TODO: pretty
					fmt.Printf("%s: %v\n", name, stat)
				}
			}
			return nil
		},
	})
	pm.AddCommand(&cobra.Command{
		Use:   "stop",
		Short: "stops one or more pm service(s)",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := client.NewHTTP()
			for _, name := range args {
				if stat, err := c.StopChild(cmd.Context(), name); err != nil {
					return fmt.Errorf("failed to stop %s: %w", name, err)
				} else {
					// TODO: pretty
					fmt.Printf("%s: %v\n", name, stat)
				}
			}
			return nil
		},
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
		fmt.Println("pm not running")
		return nil
	}
	if err := c.Terminate(cmd.Context()); err != nil {
		return fmt.Errorf("failed to terminate pm daemon: %w", err)
	}
	return nil
}

func pmAdd() *cobra.Command {
	jsonFile := ""
	main := ""
	inits := []string{}
	c := &cobra.Command{
		Use:   "add [name]",
		Short: "add a service to the pm daemon",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var child api.Child
			if len(args) > 0 {
				child.Name = args[0]
			}
			if len(main) > 0 {
				mainArgs := strings.Fields(main)
				child.Main.Cmd = mainArgs[0]
				child.Main.Args = mainArgs[1:]
			}
			for _, init := range inits {
				var ex api.Exec
				initArgs := strings.Fields(init)
				ex.Cmd = initArgs[0]
				ex.Args = initArgs[1:]
				child.Init = append(child.Init, ex)
			}
			if jsonFile != "" {
				if content, err := os.ReadFile(jsonFile); err != nil {
					return err
				} else if err := json.Unmarshal(content, &child); err != nil {
					return err
				}
			}
			// TODO: validate
			c := client.NewHTTP()
			// ping first?
			stat, err := c.PutChild(cmd.Context(), child)
			if err != nil {
				return err
			}
			// TODO: pretty
			fmt.Println(stat)
			return nil
		},
	}
	f := c.Flags()
	f.StringVarP(&jsonFile, "json", "j", jsonFile,
		"load child definition from JSON file")
	f.StringVar(&main, "main", main,
		"main command to run (split on whitespace)")
	f.StringArrayVar(&inits, "init", inits,
		"init commands to run (split on whitespace)")
	return c
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
