package pm

import (
	"fastcat.org/go/gdev/addons"
	pmResource "fastcat.org/go/gdev/addons/pm/resource"
	"fastcat.org/go/gdev/instance"
	"fastcat.org/go/gdev/resource"
)

var addon = addons.Addon[config]{
	Config: config{
		// placeholder
	},
}

type config struct {
	// placeholder
}
type option func(*config)

func Configure(opts ...option) {
	addon.CheckNotInitialized()
	for _, o := range opts {
		o(&addon.Config)
	}

	addon.RegisterIfNeeded(addons.Definition{
		Name: "pm",
		Description: func() string {
			return "Process manager daemon"
		},
		Initialize: initialize,
	})
}

func initialize() error {
	instance.AddCommandBuilders(pmCmd)
	resource.AddContextEntry(pmResource.NewPMClient)
	return nil
}
