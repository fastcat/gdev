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
	if config != nil {
		panic(errors.New("addon already enabled"))
	}
	internal.CheckCanCustomize()
	cfg := addonConfig{
		// contextName defaults to a late bind to the app name
		namespace: namespace(apiCoreV1.NamespaceDefault),
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
	contextName string
	namespace   namespace
}

type option func(*addonConfig)

func WithContext(name string) option {
	return func(ac *addonConfig) {
		ac.contextName = name
	}
}
func WithNamespace(name string) option {
	return func(ac *addonConfig) {
		ac.namespace = namespace(name)
	}
}
func (c *addonConfig) ContextName() string {
	internal.CheckLockedDown()
	if c.contextName != "" {
		return c.contextName
	}
	return instance.AppName()
}

// precise type so we can bind it into the resource context
type namespace string

func requireEnabled() {
	internal.CheckLockedDown()
	if config == nil {
		panic("k8s addon not enabled")
	}
}
