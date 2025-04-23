package cmd

import (
	"errors"
	"fmt"
	"os"
)

func Main() {
	if err := Root().Execute(); err != nil {
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
