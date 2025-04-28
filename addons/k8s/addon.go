package k8s

import (
	"context"
	"errors"

	"fastcat.org/go/gdev/addons"
	"fastcat.org/go/gdev/instance"
	"fastcat.org/go/gdev/internal"
	"fastcat.org/go/gdev/resource"
	apiCoreV1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

var config *addonConfig

func Enable(opts ...option) {
	internal.CheckCanCustomize()
	if config != nil {
		panic(errors.New("addon already enabled"))
	}
	cfg := addonConfig{
		contextName: instance.AppName,
		namespace:   namespace(apiCoreV1.NamespaceDefault),
	}
	for _, o := range opts {
		o(&cfg)
	}

	// register addon components
	resource.AddContextEntry(func(context.Context) (kubernetes.Interface, error) {
		return NewClient()
	})
	resource.AddContextEntry(func(ctx context.Context) (namespace, error) {
		return config.namespace, nil
	})
	// TODO: more

	config = &cfg
	addons.AddEnabled(addons.Description{
		Name: "k8s",
		Description: func() string {
			internal.CheckLockedDown()
			return "General kubernetes support, using context " +
				config.ContextName() +
				" and namespace " +
				string(config.namespace)
		},
	})
}

type addonConfig struct {
	contextName func() string
	namespace   namespace
}

type option func(*addonConfig)

func WithContext(name string) option {
	return func(ac *addonConfig) {
		ac.contextName = func() string { return name }
	}
}
func WithContextFunc(namer func() string) option {
	return func(ac *addonConfig) {
		ac.contextName = namer
	}
}
func WithNamespace(name string) option {
	return func(ac *addonConfig) {
		ac.namespace = namespace(name)
	}
}
func (c *addonConfig) ContextName() string {
	internal.CheckLockedDown()
	return c.contextName()
}

// precise type so we can bind it into the resource context
type namespace string

func requireEnabled() {
	internal.CheckLockedDown()
	if config == nil {
		panic("k8s addon not enabled")
	}
}
