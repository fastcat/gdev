package bootstrap

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"fastcat.org/go/gdev/addons"
	b_internal "fastcat.org/go/gdev/addons/bootstrap/internal"
	"fastcat.org/go/gdev/instance"
	"fastcat.org/go/gdev/internal"
)

var addon = addons.Addon[config]{
	Definition: addons.Definition{
		Name: "bootstrap",
		Description: func() string {
			return "Support for bootstrapping the local system with software installation & configuration"
		},
		// Initialize: initialize,
	},
	Config: config{
		plan: NewPlan(),
	},
}

func init() {
	addon.Definition.Initialize = initialize
}

type config struct {
	cmdFactories []cmdBuilder
	plan         *Plan
}

func Configure(opts ...Option) {
	addon.CheckNotInitialized()
	for _, o := range opts {
		o(&addon.Config)
	}

	addon.RegisterIfNeeded()
}

func initialize() error {
	cmd := RunPlanCmd(addon.Config.plan)
	cmd.Use = "bootstrap"
	pf := cmd.PersistentFlags()
	pf.BoolFunc("skip-logins", "skip logging into any accounts", func(s string) error {
		if s == "" {
			b_internal.SetDefault(skipLoginsKey, true)
		} else if v, err := strconv.ParseBool(s); err != nil {
			return err
		} else {
			b_internal.SetDefault(skipLoginsKey, v)
		}
		return nil
	})
	instance.AddCommands(cmd)

	for _, f := range addon.Config.cmdFactories {
		cmd.AddCommand(f.Build())
	}

	return nil
}

type Option func(*config)

type cmdBuilder interface {
	Build() *cobra.Command
}
type cmdFunc func() *cobra.Command

func (f cmdFunc) Build() *cobra.Command { return f() }

type staticCmd cobra.Command

func (c *staticCmd) Build() *cobra.Command { return (*cobra.Command)(c) }
func WithChildCmdBuilders(fns ...func() *cobra.Command) Option {
	return func(c *config) {
		for _, fn := range fns {
			c.cmdFactories = append(c.cmdFactories, cmdFunc(fn))
		}
	}
}

func WithChildCmds(cmds ...*cobra.Command) Option {
	return func(c *config) {
		for _, cmd := range cmds {
			c.cmdFactories = append(c.cmdFactories, (*staticCmd)(cmd))
		}
	}
}

func WithSteps(steps ...*Step) Option {
	return func(c *config) {
		c.plan.AddSteps(steps...)
	}
}

// NewDerivedPlan creates a new plan that is derived from the main plan, i.e. it
// has all of the steps copied to it, and you can then add more custom steps to
// it.
//
// You can pass exceptions, steps you want to exclude from the copy, if you want
// to create a plan that has most but not all of the normal steps. It is up to
// you to ensure this doesn't create a plan with missing dependencies that can't
// be run.
//
// Excluding a step that has a default factory may simply cause it to be
// re-added if other steps depend on it.
func NewDerivedPlan(exceptions ...string) *Plan {
	p := NewPlan()
	exSet := make(map[string]bool, len(exceptions))
	for _, e := range exceptions {
		exSet[e] = true
	}
	all := make([]*Step, 0, len(addon.Config.plan.ordered)+len(addon.Config.plan.pending))
	all = append(all, addon.Config.plan.ordered...)
	all = append(all, addon.Config.plan.pending...)
	all = internal.FilterSlice(all, func(s *Step) bool { return !exSet[s.name] })
	p.AddSteps(all...)
	return p
}

// WithAlternatePlanCmd registers a command that will run an alternate bootstrap
// plan from the given factory function.
//
// The customize parameter is optional, if given it will be called and can
// customize the command to e.g. override the short help or add long help or
// other settings.
func WithAlternatePlanCmd(
	name string,
	pf func() *Plan,
	customize func(*cobra.Command),
) Option {
	return WithChildCmdBuilders(func() *cobra.Command {
		cmd := RunPlanCmd(pf())
		cmd.Use = name
		cmd.Short = fmt.Sprintf("%s for %s", defaultCmdShort, name)
		if customize != nil {
			customize(cmd)
		}
		return cmd
	})
}
