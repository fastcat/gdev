package shx

import (
	"context"
	"errors"
	"maps"
	"os"
	"os/exec"
	"strings"
)

type Cmd struct {
	cmdAndArgs        []string
	combineExecErrors bool
	env               map[string]string
	onStarted         func(*os.Process)

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
	if len(c.env) > 0 {
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
