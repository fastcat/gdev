package gocache_gcs

import (
	"fastcat.org/go/gdev/addons"
	"fastcat.org/go/gdev/addons/gocache"
)

var addon = addons.Addon[config]{
	Definition: addons.Definition{
		Name:        "gocache-gcs",
		Description: func() string { return "Go build cache GCS remote storage" },
		Initialize:  initialize,
	},
}

type config struct {
	// placeholder
}

func initialize() error {
	return nil
}

func Configure() {
	addon.CheckNotInitialized()
	addon.RegisterIfNeeded()
	gocache.Configure(gocache.WithRemoteStorageFactory(gcsCacheFactory{}))
}
