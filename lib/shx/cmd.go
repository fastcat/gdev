package shx

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"syscall"
)

type Cmd struct {
	cmdAndArgs        []string
	envReset          bool
	combineExecErrors bool
	env               map[string]string
	onStarted         func(*os.Process)
	umask             *os.FileMode

	opts []Option
}

func New(
	name string,
	args ...string,
) *Cmd {
	return &Cmd{
		cmdAndArgs: append([]string{name}, args...),
		env:        make(map[string]string),
	}
}

func Run(
	ctx context.Context,
	cmdAndArgs []string,
	opts ...Option,
) (*Result, error) {
	return New(cmdAndArgs[0], cmdAndArgs[1:]...).With(opts...).Run(ctx)
}

// With applies options to the command.
//
// It panics if Run has already been called.
func (c *Cmd) With(opts ...Option) *Cmd {
	c.opts = append(c.opts, opts...)
	for _, opt := range opts {
		opt.apply(c)
	}
	return c
}

// Run runs the command and waits for it to finish.
//
// If the command fails to start, it returns a nil Result and the error. If the
// command starts but exits with an error code, the error will be in the Result.
// This behavior can be overridden with an option to copy the Result error to
// the top level error, if the caller doesn't care about the distinction.
func (c *Cmd) Run(ctx context.Context) (*Result, error) {
	// changing the umask requires hacks, see https://github.com/golang/go/issues/56016
	if c.umask != nil && runtime.GOOS != "linux" {
		return nil, fmt.Errorf("can only set umask on Linux due to Go limitations")
	}
	if c.umask != nil {
		var wg sync.WaitGroup
		var result *Result
		var err error
		wg.Go(func() {
			// workaround adopted from Go issue: break this thread's FS state (which
			// includes umask among others) off of the rest of the process so we can
			// set its umask without affecting other goroutines. This is irreversible,
			// so we never unlock the thread and the runtime will close it out at the
			// end of this goroutine.
			runtime.LockOSThread()
			if err = syscall.Unshare(syscall.CLONE_FS); err != nil {
				err = fmt.Errorf("failed to unshare FS state for umask change: %w", err)
				return
			}
			syscall.Umask(int(*c.umask)) // never fails
			result, err = c.run(ctx)
		})
		wg.Wait()
		return result, err
	}
	return c.run(ctx)
}

func (c *Cmd) run(ctx context.Context) (*Result, error) {
	cmd := exec.CommandContext(ctx, c.cmdAndArgs[0], c.cmdAndArgs[1:]...)
	c.applyEnv(cmd)
	var res Result
	for _, opt := range c.opts {
		opt.applyExec(cmd, &res)
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	if c.onStarted != nil {
		c.onStarted(cmd.Process)
	}
	res.exitErr = cmd.Wait()
	res.processState = cmd.ProcessState
	if err := res.execDone(); err != nil {
		// TODO: this isn't the right place to store this error
		res.exitErr = errors.Join(res.exitErr, err)
	}
	if c.combineExecErrors {
		return &res, res.exitErr
	}
	return &res, nil
}

func (c *Cmd) applyEnv(cmd *exec.Cmd) {
	if c.envReset {
		cmd.Env = make([]string, 0, len(c.env))
		for k, v := range c.env {
			cmd.Env = append(cmd.Env, k+"="+v)
		}
	} else if len(c.env) > 0 {
		curEnv := os.Environ()
		fullEnv := make(map[string]string, len(curEnv)+len(c.env))
		for _, e := range curEnv {
			name, val, _ := strings.Cut(e, "=")
			fullEnv[name] = val
		}
		maps.Copy(fullEnv, c.env)
		cmd.Env = make([]string, 0, len(fullEnv))
		for k, v := range fullEnv {
			cmd.Env = append(cmd.Env, k+"="+v)
		}
	}
}
