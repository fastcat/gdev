//go:build !linux

package server

import (
	"fmt"
	"os/exec"
	"runtime"
)

func setProcGroup(_ *exec.Cmd) {}

func terminateProcessGroup(_ int) error {
	return fmt.Errorf("process group termination not supported on %s", runtime.GOOS)
}

func killProcessGroup(_ int) error {
	return fmt.Errorf("process group kill not supported on %s", runtime.GOOS)
}
