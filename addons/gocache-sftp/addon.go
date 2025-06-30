package gocachesftp

import (
	"fastcat.org/go/gdev/addons"
	"fastcat.org/go/gdev/addons/gocache"
)

var addon = addons.Addon[config]{
	Definition: addons.Definition{
		Name:        "gocache-sftp",
		Description: func() string { return "Go build cache SFTP remote storage" },
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
	gocache.Configure(gocache.WithRemoteStorageFactory(sftpCacheFactory{}))
}
