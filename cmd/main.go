package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"fastcat.org/go/gdev/addons"
	"fastcat.org/go/gdev/internal"
	"fastcat.org/go/gdev/lib/config"
)

func Main() {
	addons.Initialize()
	internal.LockCustomizations()
	if err := config.Initialize(); err != nil {
		// TODO: let caller say if this should be fatal?
		fmt.Fprintf(os.Stderr, "Error initializing config: %v\n", err)
	}

	ctx := context.Background()
	// hook ctrl-c to context cancel
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()
	if err := Root().ExecuteContext(ctx); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		ec := 1
		var ece ExitCodeErr
		if errors.As(err, &ece) {
			ec = ece.ExitCode()
		}
		os.Exit(ec) //nolint:forbidigo // entrypoint
	}
}

// ExitCodeErr is an interface that can be implemented by errors
// to provide a custom exit code when the program exits due to them.
type ExitCodeErr interface {
	// ExitCode returns the exit code to use when the program exits due to this
	// error.
	ExitCode() int
}
