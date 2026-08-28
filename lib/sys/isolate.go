package sys

import (
	"context"
	"os"
	"time"
)

type IsolateUsage struct {
	User   time.Duration
	System time.Duration
}

func (u IsolateUsage) Total() time.Duration {
	return u.User + u.System
}

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
	Usage(
		ctx context.Context,
		group string,
	) (IsolateUsage, error)
}

// GetIsolator is initialized in platform-specific files, and should generally
// be the result of [sync.OnceValues].
var GetIsolator func() (Isolator, error)
