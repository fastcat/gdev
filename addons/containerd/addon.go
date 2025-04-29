package containerd

import (
	"context"
	"errors"

	"github.com/containerd/containerd/v2/client"

	"fastcat.org/go/gdev/addons"
	"fastcat.org/go/gdev/internal"
	"fastcat.org/go/gdev/resource"
)

var addon = addons.Addon[config]{
	Config: config{
		// TODO: the default socket is a bad choice because it requires root access
		clientAddr: "/run/containerd/containerd.sock",
	},
}

func Configure(opts ...option) {
	addon.CheckNotInitialized()
	for _, o := range opts {
		o(&addon.Config)
	}
	if addon.Config.clientAddr == "" {
		panic(errors.New("containerd addr required"))
	}

	addon.RegisterIfNeeded(addons.Definition{
		Name: "containerd",
		Description: func() string {
			internal.CheckLockedDown()
			return "General containerd support, using socket " + addon.Config.clientAddr
		},
		Initialize: initialize,
	})
}

func initialize() error {
	resource.AddContextEntry(func(context.Context) (*client.Client, error) {
		return NewClient()
	})

	// TODO: image puller infrastructure

	addon.Initialized()
	return nil
}

type config struct {
	clientAddr string
	clientOpts []client.Opt
}

type option func(*config)

func WithAddress(addr string) option {
	return func(ac *config) {
		ac.clientAddr = addr
	}
}
func WithOpts(opts ...client.Opt) option {
	return func(ac *config) {
		ac.clientOpts = append(ac.clientOpts, opts...)
	}
}
