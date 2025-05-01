package main

import (
	"fastcat.org/go/gdev/addons/bootstrap"
	"fastcat.org/go/gdev/addons/containerd"
	"fastcat.org/go/gdev/addons/docker"
	"fastcat.org/go/gdev/addons/k3s"
	"fastcat.org/go/gdev/addons/k8s"
	"fastcat.org/go/gdev/addons/postgres"
	"fastcat.org/go/gdev/addons/valkey"
	"fastcat.org/go/gdev/cmd"
)

// Normally you want to build your own wrapper around gdev to register your
// custom services and commands.
func main() {
	// enable all addons we can in the main build so everything gets compiled, etc.

	bootstrap.Configure()  // many will tweak it
	k8s.Configure()        // k3s will tweak it
	containerd.Configure() // k3s will tweak it
	docker.Configure()     // k3s would tweak it if we told k3s to use docker
	k3s.Configure(
		k3s.WithProvider(containerd.K3SProvider()),
	)
	postgres.Configure(
		postgres.WithService(),
	)
	valkey.Configure(
		valkey.WithService(
			valkey.WithConfig(
				// don't use too much memory in a dev demo setup
				"maxmemory 100mb",
				// evict anything to stay below the limit, on an LRU basis
				"maxmemory-policy allkeys-lru",
			),
		),
	)

	cmd.Main()
}
