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
	"fastcat.org/go/gdev/stack"
)

// Normally you want to build your own wrapper around gdev to register your
// custom services and commands.
func main() {
	// enable all addons we can in the main build so everything gets compiled, etc.

	// the steps for apt update and apt install should be added automatically due
	// to having packages to install
	bootstrap.Configure(bootstrap.WithSteps(bootstrap.Step(
		"Select Docker packages",
		func(ctx *bootstrap.Context) error {
			bootstrap.AddAptPackages(ctx, "docker.io")
			return nil
		},
		// this one is optional (default step adding will pull it in if absent due
		// to apt install referencing it), having it here makes for a more logical
		// ordering where we "choose" which docker packages to install after
		// updating the available list.
		bootstrap.WithAfter(bootstrap.StepNameAptUpdate),
		bootstrap.WithBefore(bootstrap.StepNameAptInstall),
	)))

	k8s.Configure()        // k3s will tweak it
	containerd.Configure() // k3s will tweak it
	docker.Configure()     // k3s would tweak it if we told k3s to use docker
	k3s.Configure(
		k3s.WithProvider(containerd.K3SProvider()),
	)

	stack.AddService(postgres.Service())
	stack.AddService(valkey.Service())

	cmd.Main()
}
