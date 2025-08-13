package template

import "fastcat.org/go/gdev/addons"

// addon describes the addon provided by this package.
//
// Do NOT export this variable.
var addon = addons.Addon[config]{
	Definition: addons.Definition{
		Name: "template",
		Description: func() string {
			return "Template addon for creating new addons"
		},
		// Initialize: initialize, // initialized below to avoid circular dependency
	},
	Config: config{
		// Initialize your addon configuration here
	},
}

func init() {
	addon.Definition.Initialize = initialize
}

type config struct {
	// Add fields for your addon configuration here
}

type option func(*config)

func Configure(opts ...option) {
	addon.CheckNotInitialized()
	for _, o := range opts {
		o(&addon.Config)
	}

	// If your addon depends on other addons, configure them here.
	// Declare [sync.OnceFunc] values in package scope and call them here if you
	// need to do non-idempotent addon configurations like adding bootstrap steps.

	addon.RegisterIfNeeded()
}

func initialize() error {
	// Perform any initialization logic for your addon here. At this point the
	// configuration of all addons is frozen, but you can still do other
	// customizations like adding stack services, stack resource context entries,
	// instance commands, etc.
	return nil
}

// Define any WithFoo functions returning [option] funcs for consumers to
// configure your addon. These option functions should setup your config object,
// but not put any configuration into play yet. That should not happen until
// [Configure] or [initialize] is called, depending on context.
