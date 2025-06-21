package docker

import (
	"context"

	"github.com/docker/docker/client"

	"fastcat.org/go/gdev/addons"
	"fastcat.org/go/gdev/internal"
	"fastcat.org/go/gdev/resource"
)

var addon = addons.Addon[config]{
	Definition: addons.Definition{
		Name: "docker",
		Description: func() string {
			internal.CheckLockedDown()
			return "General docker support"
		},
		Initialize: initialize,
	},
	Config: config{
		// placeholder
	},
}

func Configure(opts ...option) {
	addon.CheckNotInitialized()
	for _, o := range opts {
		o(&addon.Config)
	}

	configureBootstrap()

	addon.RegisterIfNeeded()
}

func initialize() error {
	resource.AddContextEntry(func(context.Context) (client.APIClient, error) {
		return NewClient()
	})
	return nil
}

type config struct {
	// TODO: actually support some options
}

type option func(*config)
