package valkey

import (
	"sync"

	"fastcat.org/go/gdev/addons"
	"fastcat.org/go/gdev/addons/bootstrap"
	"fastcat.org/go/gdev/internal"
	"fastcat.org/go/gdev/stack"
)

var addon addons.Addon[config]

type config struct {
	enableService bool
	svcOpts       []valkeySvcOpt
}
type option func(*config)

func Configure(opts ...option) {
	addon.CheckNotInitialized()
	for _, o := range opts {
		o(&addon.Config)
	}

	configureBootstrap()

	addon.RegisterIfNeeded(addons.Definition{
		Name: "valkey",
		Description: func() string {
			internal.CheckLockedDown()
			return "General valkey support"
		},
		Initialize: initialize,
	})
}

// WithService causes a valkey server instance to be added to the stack, using
// the given options.
func WithService(opts ...valkeySvcOpt) option {
	return func(c *config) {
		c.enableService = true
		c.svcOpts = append(c.svcOpts, opts...)
	}
}

var configureBootstrap = sync.OnceFunc(func() {
	bootstrap.Configure(bootstrap.WithAptPackages(
		"Select Valkey client packages",
		"valkey-tools",
	))
})

func initialize() error {
	if addon.Config.enableService {
		stack.AddService(Service(addon.Config.svcOpts...))
	}

	addon.Initialized()
	return nil
}
