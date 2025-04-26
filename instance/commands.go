package instance

import (
	"slices"

	"fastcat.org/go/gdev/internal"
	"github.com/spf13/cobra"
)

var commands []func() *cobra.Command

// Commands is a list of functions to run during app init to add additional
// commands to the Root command. They will be called from
// [fastcat.org/go/gdev/cmd/Root] during app startup.
//
// To add custom commands, use [AddCommands]
func Commands() []func() *cobra.Command {
	return slices.Clone(commands)
}

func AddCommands(cmds ...func() *cobra.Command) {
	internal.CheckCanCustomize()
	commands = append(commands, cmds...)
}
