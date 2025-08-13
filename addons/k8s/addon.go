package k8s

import (
	"context"

	apiCoreV1 "k8s.io/api/core/v1"

	"fastcat.org/go/gdev/addons"
	"fastcat.org/go/gdev/instance"
	"fastcat.org/go/gdev/internal"
	"fastcat.org/go/gdev/resource"
)

var addon = addons.Addon[config]{
	Definition: addons.Definition{
		Name: "k8s",
		// Description: describe,
		// Initialize: initialize,
	},
	Config: config{
		contextName: instance.AppName,
		namespace:   Namespace(apiCoreV1.NamespaceDefault),
	},
}

func init() {
	addon.Definition.Description = describe
	addon.Definition.Initialize = initialize
}

func Configure(opts ...option) {
	addon.CheckNotInitialized()
	for _, o := range opts {
		o(&addon.Config)
	}

	addon.RegisterIfNeeded()
}

func describe() string {
	internal.CheckLockedDown()
	return "General kubernetes support, using context " +
		addon.Config.ContextName() +
		" and namespace " +
		string(addon.Config.namespace)
}

func initialize() error {
	addon.CheckNotInitialized()
	// register addon components
	resource.AddContextEntry(func(context.Context) (Interface, error) {
		return NewClient()
	})
	resource.AddContextEntry(func(ctx context.Context) (Namespace, error) {
		return addon.Config.namespace, nil
	})
	// TODO: more

	return nil
}

type config struct {
	contextName func() string
	namespace   Namespace
}

type option func(*config)

func WithContext(name string) option {
	return func(ac *config) {
		ac.contextName = func() string { return name }
	}
}

func WithContextFunc(namer func() string) option {
	return func(ac *config) {
		ac.contextName = namer
	}
}

func WithNamespace(name string) option {
	return func(ac *config) {
		ac.namespace = Namespace(name)
	}
}

func (c *config) ContextName() string {
	internal.CheckLockedDown()
	return c.contextName()
}

// precise type so we can bind it into the resource context
type Namespace string
