package main

import (
	"fastcat.org/go/gdev/addons/bootstrap"
	"fastcat.org/go/gdev/addons/docker"
	"fastcat.org/go/gdev/addons/k3s"
	"fastcat.org/go/gdev/cmd"
)

// Normally you want to build your own wrapper around gdev to register your
// custom services and commands.
func main() {
	// enable all addons we can in the main build so everything gets compiled, etc.

	bootstrap.Enable()
	// k8s.Enable() // enabled by k3s
	// containerd.Enable() // enabled by k3s, which will customize its socket path
	docker.Enable()
	k3s.Enable()

	cmd.Main()
}
