package server

import (
	"os/exec"
	"syscall"
)

// setProcGroup configures the command to start in a new process group,
// enabling group-wide signal delivery via negative PID.
func setProcGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// terminateProcessGroup sends SIGTERM to the entire process group.
func terminateProcessGroup(pid int) error {
	return syscall.Kill(-pid, syscall.SIGTERM)
}

// killProcessGroup sends SIGKILL to the entire process group.
func killProcessGroup(pid int) error {
	return syscall.Kill(-pid, syscall.SIGKILL)
}
