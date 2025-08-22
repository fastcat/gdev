package sys

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"fastcat.org/go/gdev/instance"
	"fastcat.org/go/gdev/lib/shx"
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

func SudoReaderIfNecessary(ctx context.Context, fn string, allowPrompt bool) (io.ReadCloser, error) {
	var reader io.ReadCloser
	var err1 error
	if reader, err1 = os.Open(fn); err1 == nil {
		return reader, nil
	} else if errors.Is(err1, os.ErrNotExist) {
		// don't need root to confirm the same result
		return nil, err1
	}

	var err2 error
	if reader, err2 = SudoReader(ctx, fn, allowPrompt); err2 != nil {
		return nil, errors.Join(err1, err2)
	}
	return reader, nil
}

func ReadFileAsRoot(ctx context.Context, fn string, allowPrompt bool) ([]byte, error) {
	r, err := SudoReaderIfNecessary(ctx, fn, allowPrompt)
	if err != nil {
		// TODO: may double-mention the filename
		return nil, fmt.Errorf("failed to open %s: %w", fn, err)
	}
	defer r.Close() // nolint:errcheck
	content, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", fn, err)
	}
	if err = r.Close(); err != nil {
		return nil, fmt.Errorf("failed to close %s: %w", fn, err)
	}
	return content, nil
}

func WriteFileAsRoot(ctx context.Context, fn string, content io.Reader, mode os.FileMode) error {
	// TODO: maybe write to a temp file and use `install` instead?

	dir := filepath.Dir(fn)
	if _, err := os.Stat(dir); err != nil {
		// give dirs we create sane permissions
		dm := (mode & 0o775) | 0o700
		if _, err := shx.Run(
			ctx,
			[]string{"mkdir", "-p", dir, "-m", fmt.Sprintf("%04o", dm)},
			shx.WithSudo(fmt.Sprintf("mkdir %s", dir)),
			shx.WithCombinedError(),
		); err != nil {
			return err
		}
	}

	// TODO: make sure there aren't nasty symlinks involved here and such

	if _, err := shx.Run(
		ctx,
		[]string{"tee", fn},
		shx.WithSudo(fmt.Sprintf("write %s", fn)),
		shx.FeedStdin(content),
		shx.WithCombinedError(),
	); err != nil {
		return err
	}

	// TODO: we'd like to set the file mode atomically with creating it, but
	// that's tricky without extra dependencies

	if _, err := shx.Run(
		ctx,
		[]string{"chmod", fmt.Sprintf("%04o", mode), fn},
		shx.WithSudo(fmt.Sprintf("set permissions on %s", fn)),
		shx.PassStderr(),
		shx.WithCombinedError(),
	); err != nil {
		return err
	}

	return nil
}
