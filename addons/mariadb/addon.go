package mariadb

import (
	"sync"

	"fastcat.org/go/gdev/addons"
	"fastcat.org/go/gdev/addons/bootstrap"
	"fastcat.org/go/gdev/addons/bootstrap/apt"
	"fastcat.org/go/gdev/internal"
	"fastcat.org/go/gdev/stack"
)

var addon = addons.Addon[config]{
	Definition: addons.Definition{
		Name: "mariadb",
		Description: func() string {
			internal.CheckLockedDown()
			return "General mariadb support"
		},
		// Initialize: initialize,
	},
	Config: config{
		// placeholder
	},
}

func init() {
	addon.Definition.Initialize = initialize
}

type config struct {
	enableService bool
	svcOpts       []svcOpt
}
type option func(*config)

func Configure(opts ...option) {
	addon.CheckNotInitialized()
	for _, o := range opts {
		o(&addon.Config)
	}

	configureBootstrap()

	addon.RegisterIfNeeded()
}

// WithService causes a mariadb server instance to be added to the stack, using
// the given options.
func WithService(opts ...svcOpt) option {
	return func(c *config) {
		c.enableService = true
		c.svcOpts = append(c.svcOpts, opts...)
	}
}

var configureBootstrap = sync.OnceFunc(func() {
	bootstrap.Configure(apt.WithPackages(
		"Select MariaDB client packages",
		"mariadb-client",
	))
})

func initialize() error {
	if addon.Config.enableService {
		stack.AddInfrastructure(Service(addon.Config.svcOpts...))
	}

	return nil
}
