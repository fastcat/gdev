package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"fastcat.org/go/gdev/addons"
	"fastcat.org/go/gdev/instance"
	"fastcat.org/go/gdev/internal"
)

func Root() *cobra.Command {
	var longDesc strings.Builder
	vi := instance.VersionInfo()
	fmt.Fprintf(&longDesc, "%s version %s\n", instance.AppName(), vi.MainVersion)
	versionDesc := longDesc.String()
	fmt.Fprintf(&longDesc, "Built from %s version %s (%s)\n", vi.MainModule, vi.MainRev, vi.MainRev)
	if vi.IsDebugBuild {
		fmt.Fprint(&longDesc, "Built for debugging\n")
	}
	if internal.IsDebuggerAttached() {
		fmt.Fprint(&longDesc, "Debugger is attached\n")
	}
	fmt.Fprintf(&longDesc, "Using GDev version %s\n", vi.GDevVersion)
	if ao := addons.Enabled(); len(ao) > 0 {
		fmt.Fprint(&longDesc, "Built with addons:")
		for i, a := range ao {
			if i == 0 {
				fmt.Fprint(&longDesc, " ")
			} else {
				fmt.Fprint(&longDesc, ", ")
			}
			fmt.Fprint(&longDesc, a.Name)
		}
	}

	root := &cobra.Command{
		Use:           instance.AppName(),
		Long:          versionDesc,
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       instance.Version(),
	}
	root.AddCommand(&cobra.Command{
		Use:                   "version",
		Short:                 "Detailed version information",
		Args:                  cobra.NoArgs,
		DisableFlagsInUseLine: true,
		RunE: func(*cobra.Command, []string) error {
			_, err := fmt.Println(longDesc.String())
			return err
		},
	})
	root.AddCommand(instance.Commands()...)
	return root
}
