package server

import (
	"context"
	"os"
)

type isolator interface {
	isolateProcess(
		ctx context.Context,
		name string,
		process *os.Process,
	) (group string, err error)
	cleanup(
		ctx context.Context,
		group string,
	) error
}

// getIsolator is initialized in platform-specific files, and should generally
// be the result of [sync.OnceValues].
var getIsolator func() (isolator, error)
