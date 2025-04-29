package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"fastcat.org/go/gdev/addons"
	"fastcat.org/go/gdev/instance"
)

func Root() *cobra.Command {
	var longDesc strings.Builder
	fmt.Fprintf(&longDesc, "%s version %s\n", instance.AppName(), instance.Version())
	vi := instance.VersionInfo()
	fmt.Fprintf(&longDesc, "Built from %s version %s (%s)\n", vi.MainModule, vi.MainRev, vi.MainRev)
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
		Long:          longDesc.String(),
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       instance.Version(),
	}
	root.AddCommand(instance.Commands()...)
	return root
}
