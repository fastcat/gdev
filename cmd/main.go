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
)

func Main() {
	addons.Initialize()
	internal.LockCustomizations()
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
		os.Exit(ec)
	}
}

type ExitCodeErr interface {
	ExitCode() int
}
