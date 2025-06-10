package nodejs

import (
	"sync"

	"fastcat.org/go/gdev/addons"
	"fastcat.org/go/gdev/addons/build"
)

var addon = addons.Addon[config]{
	Config: config{
		// TODO
	},
}

type config struct {
	// TODO
}
type option func(*config)

func Configure(opts ...option) {
	addon.CheckNotInitialized()
	for _, o := range opts {
		o(&addon.Config)
	}

	configureBuild()

	addon.RegisterIfNeeded(addons.Definition{
		Name: "nodejs",
		Description: func() string {
			return "Support for Node.js development"
		},
		Initialize: initialize,
	})
}

var configureBuild = sync.OnceFunc(func() {
	build.Configure(
		build.WithStrategy("npm", detectNPM, nil),
	)
})

func initialize() error {
	// TODO
	addon.Initialized()
	return nil
}
