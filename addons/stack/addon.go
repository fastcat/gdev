package stack

import "fastcat.org/go/gdev/addons"

// addon describes the addon provided by this package.
//
// Do NOT export this variable.
var addon = addons.Addon[config]{
	Definition: addons.Definition{
		Name: "stack",
		Description: func() string {
			return "Support for running a software stack defined of services composed of resource"
		},
		// Initialize: initialize, // initialized below to avoid circular dependency
	},
	Config: config{
		enableDefaultCommands: true,
	},
}

func init() {
	addon.Definition.Initialize = initialize
}

type config struct {
	enableDefaultCommands bool
}

type option func(*config)

func Configure(opts ...option) {
	addon.CheckNotInitialized()
	for _, o := range opts {
		o(&addon.Config)
	}

	addon.RegisterIfNeeded()
}

func initialize() error {
	registerCommands()
	return nil
}

func WithoutDefaultCommands() option {
	return func(c *config) {
		c.enableDefaultCommands = false
	}
}
