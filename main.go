package main

import (
	"fastcat.org/go/gdev/addons/containerd"
	"fastcat.org/go/gdev/addons/docker"
	"fastcat.org/go/gdev/addons/k8s"
	"fastcat.org/go/gdev/cmd"
)

// Normally you want to build your own wrapper around gdev to register your
// custom services and commands.
func main() {
	// enable all addons we can in the main build so everything gets compiled, etc.
	k8s.Enable()
	// TODO: containerd requires a socket path, this will get turned on with k3s support
	containerd.Enable()
	docker.Enable()
	cmd.Main()
}
