package gocache_http

import (
	"net/http"

	"fastcat.org/go/gdev/addons"
	"fastcat.org/go/gdev/addons/gocache"
)

var addon = addons.Addon[config]{
	Definition: addons.Definition{
		Name:        "gocache-http",
		Description: func() string { return "Go build cache HTTP remote storage" },
		Initialize:  initialize,
	},
}

type config struct {
	auth      Authorizer
	putMethod string
}
type option func(*config)

type Authorizer interface {
	// Authorize updates the request with any necessary headers or other changes
	// to authorize it against the backend.
	Authorize(req *http.Request) error
}

func Configure(opts ...option) {
	addon.CheckNotInitialized()
	for _, o := range opts {
		o(&addon.Config)
	}
	addon.RegisterIfNeeded()
	gocache.Configure(gocache.WithRemoteStorageFactory(factory{}))
}

func initialize() error {
	return nil
}
