package gocache_s3

import (
	"fmt"

	"fastcat.org/go/gdev/addons"
	"fastcat.org/go/gdev/addons/gocache"
)

var addon = addons.Addon[config]{
	Definition: addons.Definition{
		Name:        "gocache-s3",
		Description: func() string { return "Go build cache S3 remote storage" },
		// Initialize:  initialize,
	},
}

func init() {
	addon.Definition.Initialize = initialize
}

type config struct {
	region string
}
type option func(*config)

func WithRegion(region string) option {
	if region == "" {
		panic("region must not be empty")
	}
	return func(c *config) {
		c.region = region
	}
}

func initialize() error {
	if addon.Config.region == "" {
		return fmt.Errorf("gocache-s3: region must be specified")
	}
	return nil
}

func Configure(opts ...option) {
	addon.CheckNotInitialized()
	for _, o := range opts {
		o(&addon.Config)
	}
	addon.RegisterIfNeeded()
	gocache.Configure(gocache.WithRemoteStorageFactory(factory{}))
}
