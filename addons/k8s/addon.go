package k8s

import (
	"context"
	"errors"

	"fastcat.org/go/gdev/addons"
	"fastcat.org/go/gdev/instance"
	"fastcat.org/go/gdev/internal"
	"fastcat.org/go/gdev/resource"
	"k8s.io/client-go/kubernetes"
)

var config *addonConfig

func Enable(opts ...option) {
	if config != nil {
		panic(errors.New("addon already enabled"))
	}
	internal.CheckCanCustomize()
	var cfg addonConfig
	for _, o := range opts {
		o(&cfg)
	}

	// register addon components
	resource.AddContextEntry(func(context.Context) (kubernetes.Interface, error) {
		return NewClient()
	})
	// TODO: more

	config = &cfg
	addons.AddEnabled(addons.Description{
		Name: "k8s",
		Description: func() string {
			internal.CheckLockedDown()
			return "General kubernetes support, using context " + config.ContextName()
		},
	})
}

type addonConfig struct {
	contextName string
}

type option func(*addonConfig)

func WithContext(name string) option {
	return func(ac *addonConfig) {
		ac.contextName = name
	}
}
func (c *addonConfig) ContextName() string {
	internal.CheckLockedDown()
	if c.contextName != "" {
		return c.contextName
	}
	return instance.AppName()
}
