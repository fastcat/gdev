package bootstrap

import (
	"fastcat.org/go/gdev/addons"
	"fastcat.org/go/gdev/instance"
	"github.com/spf13/cobra"
)

var addon = addons.Addon[config]{
	Config: config{
		plan: NewPlan(),
	},
}

type config struct {
	cmdFactories []cmdBuilder
	plan         *plan
}

func Configure(opts ...option) {
	addon.CheckNotInitialized()
	for _, o := range opts {
		o(&addon.Config)
	}

	addon.RegisterIfNeeded(addons.Definition{
		Name: "bootstrap",
		Description: func() string {
			return "Support for bootstrapping the local system with software installation & configuration"
		},
		Initialize: initialize,
	})
}

func initialize() error {
	cmd := &cobra.Command{
		Use:   "bootstrap",
		Args:  cobra.NoArgs,
		Short: "install & configure system dependencies",
		RunE: func(cmd *cobra.Command, args []string) error {
			return Run(cmd.Context())
		},
	}
	instance.AddCommands(cmd)
	for _, f := range addon.Config.cmdFactories {
		cmd.AddCommand(f.Build())
	}

	addon.Initialized()

	return nil
}

type option func(*config)

type cmdBuilder interface {
	Build() *cobra.Command
}
type cmdFunc func() *cobra.Command

func (f cmdFunc) Build() *cobra.Command { return f() }

type staticCmd cobra.Command

func (c *staticCmd) Build() *cobra.Command { return (*cobra.Command)(c) }
func WithChildCmdBuilders(fns ...func() *cobra.Command) option {
	return func(c *config) {
		for _, fn := range fns {
			c.cmdFactories = append(c.cmdFactories, cmdFunc(fn))
		}
	}
}

func WithChildCmds(cmds ...*cobra.Command) option {
	return func(c *config) {
		for _, cmd := range cmds {
			c.cmdFactories = append(c.cmdFactories, (*staticCmd)(cmd))
		}
	}
}

func WithSteps(steps ...*step) option {
	return func(c *config) {
		c.plan.AddSteps(steps...)
	}
}
