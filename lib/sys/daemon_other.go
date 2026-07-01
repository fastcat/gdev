//go:build !linux

package sys

import (
	"context"
	"fmt"
	"runtime"
)

// FallbackLogFileEnv is an environment variable that can be set to provide
// a fallback log file path for daemons started without systemd support.
//
// This will not be passed to the actual daemon.
const FallbackLogFileEnv = "__FALLBACK_LOG_FILE"

func StartDaemon(
	ctx context.Context,
	name string,
	path string,
	args []string,
	env map[string]string,
) error {
	return fmt.Errorf("daemon start not supported on %s", runtime.GOOS)
}
