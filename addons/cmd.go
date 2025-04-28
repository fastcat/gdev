package addons

import (
	"fmt"

	"fastcat.org/go/gdev/instance"
	"github.com/spf13/cobra"
)

func init() {
	instance.AddCommands(&cobra.Command{
		Use:   "addons",
		Short: "Describe enabled addons",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Enabled addons:")
			// TODO: github.com/jedib0t/go-pretty/v6/list
			for _, ao := range Enabled() {
				fmt.Printf("%s:\n", ao.Name)
				// TODO: wrap so extra lines stay indented?
				fmt.Printf("\t%s\n", ao.Description())
			}
			return nil
		},
	})
}
