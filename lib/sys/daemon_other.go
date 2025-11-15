//go:build !linux

package sys

import (
	"context"
	"fmt"
	"runtime"
)

func StartDaemon(
	ctx context.Context,
	name string,
	path string,
	args []string,
	env map[string]string,
) error {
	return fmt.Errorf("daemon start not supported on %s", runtime.GOOS)
}
