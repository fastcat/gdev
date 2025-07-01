package gcs

import (
	"fastcat.org/go/gdev/addons"
	"fastcat.org/go/gdev/addons/gcs/internal"
	"fastcat.org/go/gdev/resource"
)

var addon = addons.Addon[internal.Config]{
	Definition: addons.Definition{
		Name: "gcs",
		Description: func() string {
			return "Google Cloud Storage (GCS) integration"
		},
	},
}

func init() {
	addon.Definition.Initialize = initialize
}

func Configure(opts ...internal.Option) {
	addon.CheckNotInitialized()
	for _, opt := range opts {
		opt(&addon.Config)
	}
	addon.RegisterIfNeeded()
}

func initialize() error {
	if addon.Config.FakeServerImage == "" {
		addon.Config.FakeServerImage = FakeServerDefaultImage
	}
	if addon.Config.ExposedPort == 0 {
		addon.Config.ExposedPort = DefaultExposedPort
	}

	for _, hook := range addon.Config.StackHooks {
		if err := hook(&addon.Config); err != nil {
			return err
		}
	}

	resource.AddContextEntry(NewEmulatorClient)

	return nil
}

const (
	FakeServerDefaultImage = "fsouza/fake-gcs-server"
	DefaultExposedPort     = 4443 // FIXME
)
