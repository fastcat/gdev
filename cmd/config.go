package cmd

import (
	"fmt"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"fastcat.org/go/gdev/instance"
	"fastcat.org/go/gdev/lib/config"
	"fastcat.org/go/gdev/service"
	"fastcat.org/go/gdev/stack"
)

func Config() *cobra.Command {
	cfg := &cobra.Command{
		Use: "config",
		// just a parent for other commands
	}
	cfg.AddCommand(&cobra.Command{
		Use:  "mode",
		Args: cobra.RangeArgs(0, 2),
		ValidArgsFunction: func(
			_ *cobra.Command,
			args []string,
			toComplete string,
		) ([]string, cobra.ShellCompDirective) {
			var candidates []string
			if len(args) == 0 {
				allServices := append(stack.AllInfrastructure(), stack.AllServices()...)
				candidates = make([]string, 0, len(allServices))
				for _, s := range allServices {
					candidates = append(candidates, s.Name())
				}
			} else if len(args) == 1 {
				candidates = service.ValidModeNames()
			}
			candidates = slices.DeleteFunc(candidates, func(candidate string) bool {
				return !strings.HasPrefix(candidate, toComplete)
			})
			return candidates, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			allServices := append(stack.AllInfrastructure(), stack.AllServices()...)
			findSvc := func(name string) (service.Service, bool) {
				if i := slices.IndexFunc(allServices, func(s service.Service) bool {
					return s.Name() == name
				}); i < 0 {
					return nil, false
				} else {
					return allServices[i], true
				}
			}
			if len(args) == 0 {
				// print all configured service modes
				modes := service.ConfiguredModes()
				if len(modes) == 0 {
					fmt.Println("All services will run in default mode")
					return nil
				}
				for s, m := range modes {
					if svc, ok := findSvc(s); ok {
						if m != service.ModeDisabled && m != service.ModeDebug && !svc.HasModal(m) {
							fmt.Printf("WARNING: service %s does not have support for %s mode\n", s, m)
						}
						fmt.Printf("Service %s will run in %s mode\n", s, m)
					} else {
						fmt.Printf("WARNING: unknown service %q configured for %s mode\n", s, m)
					}
				}
				fmt.Println("All other services will run in default mode")
				return nil
			}
			currentMode := service.ConfiguredMode(args[0])
			// print a warning if it's not a known service name
			svc, ok := findSvc(args[0])
			if !ok {
				fmt.Printf("WARNING: unknown service name %q\n", args[0])
			}

			if len(args) == 1 {
				fmt.Printf("Current mode for service %s: %s\n", args[0], currentMode)
				if currentMode != service.ModeDisabled &&
					currentMode != service.ModeDebug &&
					svc != nil && !svc.HasModal(currentMode) {
					fmt.Printf("WARNING: service %s does not have support for %s mode\n", args[0], currentMode)
				}
				return nil
			}

			newMode, ok := service.ParseMode(args[1])
			if !ok {
				return fmt.Errorf("invalid mode %q for service %q", args[1], args[0])
			} else if newMode != service.ModeDebug && newMode != service.ModeDisabled && !svc.HasModal(newMode) {
				return fmt.Errorf("service %s does not have support for %s mode", args[0], newMode)
			}
			if newMode == currentMode {
				fmt.Printf("service %s already set for %s mode\n", args[0], newMode)
				return nil
			}
			service.SetMode(args[0], newMode)
			if err := config.Save(); err != nil {
				return fmt.Errorf("failed to save config: %w", err)
			}
			fmt.Printf("service %s will run in %s mode on next start\n", args[0], newMode)
			return nil
		},
	})
	return cfg
}

func init() {
	instance.AddCommandBuilders(Config)
}
