package sys

import (
	"context"
	"os"
)

type Isolator interface {
	Isolate(
		ctx context.Context,
		name string,
		process *os.Process,
	) (group string, err error)
	Cleanup(
		ctx context.Context,
		group string,
	) error
}

// GetIsolator is initialized in platform-specific files, and should generally
// be the result of [sync.OnceValues].
var GetIsolator func() (Isolator, error)
