package instance

import (
	"github.com/spf13/cobra"

	"fastcat.org/go/gdev/internal"
)

type builder interface {
	Cmd() *cobra.Command
}

var builders []builder

// Commands gets a list of additional commands to add to the
// [fastcat.org/go/gdev/cmd/Root] command during app startup.
//
// This will panic if called during the customization phase before the main app
// startup.
//
// To add custom commands, use [AddCommandBuilders] or [AddCommands]
func Commands() []*cobra.Command {
	internal.CheckLockedDown()
	ret := make([]*cobra.Command, 0, len(builders))
	for _, b := range builders {
		ret = append(ret, b.Cmd())
	}
	return ret
}

func AddCommandBuilders(fns ...func() *cobra.Command) {
	internal.CheckCanCustomize()
	for _, b := range fns {
		builders = append(builders, cmdFunc(b))
	}
}

func AddCommands(cmds ...*cobra.Command) {
	internal.CheckCanCustomize()
	for _, c := range cmds {
		builders = append(builders, (*staticCmd)(c))
	}
}

type staticCmd cobra.Command

func (c *staticCmd) Cmd() *cobra.Command { return (*cobra.Command)(c) }

type cmdFunc func() *cobra.Command

func (c cmdFunc) Cmd() *cobra.Command { return c() }
