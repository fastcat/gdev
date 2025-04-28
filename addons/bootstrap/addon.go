package bootstrap

import (
	"errors"
	"sync/atomic"

	"fastcat.org/go/gdev/instance"
	"fastcat.org/go/gdev/internal"
	"github.com/spf13/cobra"
)

var enabled atomic.Bool

func Enable() {
	internal.CheckCanCustomize()
	if !enabled.CompareAndSwap(false, true) {
		panic(errors.New("addon already enabled"))
	}

	instance.AddCommands(&cobra.Command{
		Use:   "bootstrap",
		Args:  cobra.NoArgs,
		Short: "install & configure system dependencies",
		RunE: func(cmd *cobra.Command, args []string) error {
			return Run(cmd.Context())
		},
	})
}
