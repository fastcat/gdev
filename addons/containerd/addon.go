package containerd

import (
	"context"
	"errors"

	"fastcat.org/go/gdev/addons"
	"fastcat.org/go/gdev/internal"
	"fastcat.org/go/gdev/resource"
	"github.com/containerd/containerd/v2/client"
)

var config *addonConfig

func Enable(opts ...option) {
	internal.CheckCanCustomize()
	if config != nil {
		panic(errors.New("addon already enabled"))
	}
	cfg := addonConfig{
		// TODO: the default socket is a bad choice because it requires root access
		clientAddr: "/run/containerd/containerd.sock",
	}
	for _, o := range opts {
		o(&cfg)
	}
	if cfg.clientAddr == "" {
		panic(errors.New("containerd addr required"))
	}

	resource.AddContextEntry(func(context.Context) (*client.Client, error) {
		return NewClient()
	})
	// TODO: image puller infrastructure

	config = &cfg
	addons.AddEnabled(addons.Description{
		Name: "containerd",
		Description: func() string {
			internal.CheckLockedDown()
			return "General containerd support, using socket " + config.clientAddr
		},
	})
}

type addonConfig struct {
	clientAddr string
	clientOpts []client.Opt
}

type option func(*addonConfig)

func WithAddress(addr string) option {
	return func(ac *addonConfig) {
		ac.clientAddr = addr
	}
}
func WithOpts(opts ...client.Opt) option {
	return func(ac *addonConfig) {
		ac.clientOpts = append(ac.clientOpts, opts...)
	}
}
