package pm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"

	"github.com/jedib0t/go-pretty/v6/list"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"

	"fastcat.org/go/gdev/addons/pm/api"
	"fastcat.org/go/gdev/addons/pm/client"
	"fastcat.org/go/gdev/addons/pm/server"
)

func pmCmd() *cobra.Command {
	pm := &cobra.Command{
		Use:   "pm",
		Short: "starts the pm daemon if it isn't already running",
		Args:  cobra.NoArgs,
		RunE:  pmAutoStart,
	}
	pm.AddCommand(&cobra.Command{
		Use:   "status [service...]",
		Short: "show pm services",
		Long: "With no args, shows a summary table for all services. " +
			"With one or more args, shows details of those services",
		RunE: PMStatus,
	})
	pm.AddCommand(&cobra.Command{
		Use:   "terminate",
		Short: "terminate pm daemon and any children",
		Args:  cobra.NoArgs,
		RunE:  PMTerminate,
	})

	pm.AddCommand(pmAdd())

	pm.AddCommand(&cobra.Command{
		Use:   "start <name...>>",
		Short: "starts one or more pm service(s)",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := client.NewHTTP()
			for _, name := range args {
				if stat, err := c.StartChild(cmd.Context(), name); err != nil {
					return fmt.Errorf("failed to start %s: %w", name, err)
				} else {
					PrettyChildStatus(stat, os.Stdout)
				}
			}
			return nil
		},
	})

	stop := &cobra.Command{
		Use:   "stop <name...>",
		Short: "stops one or more pm service(s)",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := client.NewHTTP()
			for _, name := range args {
				if stat, err := c.StopChild(cmd.Context(), name); err != nil {
					return fmt.Errorf("failed to stop %s: %w", name, err)
				} else {
					PrettyChildStatus(stat, os.Stdout)
				}
			}
			return nil
		},
	}
	pm.AddCommand(stop)
	stop.AddCommand(&cobra.Command{
		Use:   "group <name...>",
		Short: "stops all pm services in the given group(s)",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := client.NewHTTP()
			children, err := c.Summary(cmd.Context())
			if err != nil {
				return err
			}
			for _, child := range children {
				// treat no group as the empty group
				g := child.Annotations[api.AnnotationGroup]
				if !slices.Contains(args, g) {
					continue
				}
				if stat, err := c.StopChild(cmd.Context(), child.Name); err != nil {
					return fmt.Errorf("failed to stop %s: %w", child.Name, err)
				} else {
					PrettyChildStatus(stat, os.Stdout)
				}
			}
			return nil
		},
	})

	pm.AddCommand(&cobra.Command{
		Use:     "remove <name...>",
		Aliases: []string{"rm"},
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := client.NewHTTP()
			for _, name := range args {
				if stat, err := c.DeleteChild(cmd.Context(), name); err != nil {
					return fmt.Errorf("failed to remove %s: %w", name, err)
				} else {
					PrettyChildStatus(stat, os.Stdout)
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

func PMStatus(cmd *cobra.Command, args []string) error {
	c := client.NewHTTP()
	if err := c.Ping(cmd.Context()); err != nil {
		return fmt.Errorf("pm is not running: %w", err)
	}

	if len(args) == 0 {
		return PMStatusTable(cmd.Context(), c)
	}
	return PMStatusDetail(cmd.Context(), c, args...)
}

func PMStatusTable(ctx context.Context, client api.API) error {
	summary, err := client.Summary(ctx)
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
			h = healthEmoji(*c.Healthy)
		}
		tw.AppendRow(table.Row{c.Name, c.State, c.Pid, h})
	}
	tw.Render()
	return nil
}

func PMStatusDetail(ctx context.Context, client api.API, names ...string) error {
	for _, name := range names {
		stat, err := client.Child(ctx, name)
		if err != nil {
			return fmt.Errorf("failed to get child %s status: %w", name, err)
		}
		PrettyChildStatus(stat, os.Stdout)
	}
	return nil
}

func pmAutoStart(cmd *cobra.Command, _ []string) error {
	return client.AutoStart(cmd.Context(), client.NewHTTP())
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

func PrettyChildStatus(s *api.ChildWithStatus, out io.Writer) {
	l := list.NewWriter()
	l.SetOutputMirror(out)
	l.SetStyle(list.StyleConnectedLight)
	l.AppendItem("Name: " + s.Name)
	l.AppendItem("State: " + s.Status.State)
	if s.HealthCheck != nil {
		l.AppendItem("Healthy: " + healthEmoji(s.Status.Health.Healthy))
		// TODO: do somethin with LastHealthy/LastUnhealthy
	}
	renderExec := func(e api.Exec, s api.ExecStatus) {
		l.AppendItem(strings.Join(append([]string{e.Cmd}, e.Args...), " "))
		// TODO: Cwd, Env
		l.Indent()
		switch s.State {
		case api.ExecNotStarted:
			if s.StartErr != "" {
				l.AppendItem(fmt.Sprintf("Failed to start: %s", s.StartErr))
			}
		case api.ExecRunning:
			l.AppendItem(fmt.Sprintf("Running, pid: %d", s.Pid))
		case api.ExecEnded:
			if s.ExitCode == 0 {
				l.AppendItem("Done")
			} else {
				l.AppendItem(fmt.Sprintf("Failed, exit code: %d", s.ExitCode))
			}
		case api.ExecStopping:
			l.AppendItem(fmt.Sprintf("Stopping, pid: %d", s.Pid))
		}
		l.UnIndent()
	}
	if len(s.Init) != 0 {
		l.AppendItem("Init")
		l.Indent()
		for i, init := range s.Init {
			renderExec(init, s.Status.Init[i])
		}
		l.UnIndent()
		l.AppendItem("Main")
		l.Indent()
		renderExec(s.Main, s.Status.Main)
		l.UnIndent()
	}
	l.Render()
}

func healthEmoji(value bool) string {
	if value {
		return "👍"
	} else {
		return "❌"
	}
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
			PrettyChildStatus(stat, os.Stdout)
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
