package docker

import (
	"context"
	"errors"

	"fastcat.org/go/gdev/addons"
	"fastcat.org/go/gdev/internal"
	"fastcat.org/go/gdev/resource"
	"github.com/docker/docker/client"
)

var config *addonConfig

func Enable(opts ...option) {
	internal.CheckCanCustomize()
	if config != nil {
		panic(errors.New("addon already enabled"))
	}
	cfg := addonConfig{}
	for _, o := range opts {
		o(&cfg)
	}

	resource.AddContextEntry(func(context.Context) (client.APIClient, error) {
		return NewClient()
	})

	config = &cfg
	addons.AddEnabled(addons.Description{
		Name: "docker",
		Description: func() string {
			internal.CheckLockedDown()
			return "General docker support"
		},
	})

}

type addonConfig struct {
	// TODO: actually support some options
}

type option func(*addonConfig)
