package docker

import (
	"context"

	"fastcat.org/go/gdev/addons"
	"fastcat.org/go/gdev/internal"
	"fastcat.org/go/gdev/resource"
	"github.com/docker/docker/client"
)

var addon addons.Addon[config]

func Configure(opts ...option) {
	addon.CheckNotInitialized()
	for _, o := range opts {
		o(&addon.Config)
	}

	addon.RegisterIfNeeded(addons.Definition{
		Name: "docker",
		Description: func() string {
			internal.CheckLockedDown()
			return "General docker support"
		},
		Initialize: initialize,
	})

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
