package main

import (
	"fastcat.org/go/gdev/addons/bootstrap"
	"fastcat.org/go/gdev/addons/build"
	"fastcat.org/go/gdev/addons/containerd"
	"fastcat.org/go/gdev/addons/docker"
	"fastcat.org/go/gdev/addons/gcs"
	gcs_k8s "fastcat.org/go/gdev/addons/gcs/k8s"
	"fastcat.org/go/gdev/addons/gocache"
	gocachesftp "fastcat.org/go/gdev/addons/gocache-sftp"
	"fastcat.org/go/gdev/addons/golang"
	"fastcat.org/go/gdev/addons/k3s"
	"fastcat.org/go/gdev/addons/k8s"
	"fastcat.org/go/gdev/addons/nodejs"
	"fastcat.org/go/gdev/addons/pm"
	"fastcat.org/go/gdev/addons/postgres"
	"fastcat.org/go/gdev/addons/valkey"
	"fastcat.org/go/gdev/cmd"
	"fastcat.org/go/gdev/instance"
)

// Normally you want to build your own wrapper around gdev to register your
// custom services and commands.
func main() {
	instance.SetAppName("gdev")

	// enable all addons we can in the main build so everything gets compiled, etc.

	bootstrap.Configure() // many will tweak it
	pm.Configure()
	k8s.Configure()        // k3s will tweak it
	containerd.Configure() // k3s will tweak it
	docker.Configure()     // k3s would tweak it if we told k3s to use docker
	k3s.Configure(
		k3s.WithProvider(containerd.K3SProvider()),
		k3s.WithK3SArgs(
			// allow using any unprivileged port as a node port
			"--service-node-port-range=1024-65535",
		),
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
	build.Configure() // strategies will be registered by other addons
	nodejs.Configure()
	golang.Configure()
	gocache.Configure()
	gocachesftp.Configure()
	gcs.Configure(gcs_k8s.WithK8SService())

	cmd.Main()
}
