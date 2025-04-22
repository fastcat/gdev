package main

import (
	"os"

	"fastcat.org/go/gdev/cmd"
)

// Normally you want to build your own wrapper around gdev to register your
// custom services and commands.
func main() {
	if err := cmd.Main(); err != nil {
		// TODO: extract a preferred exit code from the error if we can
		os.Exit(1)
	}
}
