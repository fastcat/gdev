package main

import (
	"fastcat.org/go/gdev/addons/k8s"
	"fastcat.org/go/gdev/cmd"
)

// Normally you want to build your own wrapper around gdev to register your
// custom services and commands.
func main() {
	// enable all addons in the main build so everything gets compiled, etc.
	k8s.Enable()
	cmd.Main()
}
