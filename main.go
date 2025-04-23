package main

import (
	"fastcat.org/go/gdev/cmd"
)

// Normally you want to build your own wrapper around gdev to register your
// custom services and commands.
func main() {
	cmd.Main()
}
