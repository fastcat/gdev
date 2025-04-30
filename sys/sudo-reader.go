package sys

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"

	"fastcat.org/go/gdev/instance"
)

type sudoReader struct {
	cmd    *exec.Cmd
	cancel context.CancelFunc
	r      io.ReadCloser
}

var _ io.ReadCloser = (*sudoReader)(nil)

func SudoReader(ctx context.Context, fn string, allowPrompt bool) (*sudoReader, error) {
	ctx, cancel := context.WithCancel(ctx)
	args := []string{"cat", fn}
	if !allowPrompt {
		args = append([]string{"-n"}, args...)
	}
	cmd := exec.CommandContext(ctx, "sudo", args...)
	if allowPrompt {
		cmd.Env = os.Environ()
		cmd.Env = append(cmd.Env, fmt.Sprintf("SUDO_PROMPT=%s needs to read %s", instance.AppName(), fn))
	}
	r, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return nil, err
	}
	cmd.Stdin, cmd.Stderr = os.Stdin, os.Stderr

	if err := cmd.Start(); err != nil {
		cancel()
		// r will have been closed by Start's error handling
		return nil, err
	}
	return &sudoReader{cmd, cancel, r}, nil
}

// Close implements io.ReadCloser.
func (s *sudoReader) Close() error {
	err1 := s.r.Close()
	s.cancel() // will kill the child process
	err2 := s.cmd.Wait()
	if errors.Is(err2, context.Canceled) {
		err2 = nil
	}
	return errors.Join(err1, err2) // will be nil if no errors
}

// Read implements io.ReadCloser.
func (s *sudoReader) Read(p []byte) (n int, err error) {
	return s.r.Read(p)
}
