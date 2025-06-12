package shx

import (
	"context"
	"maps"
	"os"
	"os/exec"
	"strings"
)

type Cmd struct {
	cmd       *exec.Cmd
	envReset  bool
	env       map[string]string
	onStarted func(*os.Process)
	res       Result
}

type option func(*Cmd)

func New(
	ctx context.Context,
	name string,
	args ...string,
) *Cmd {
	c := exec.CommandContext(ctx, name, args...)
	return &Cmd{
		cmd: c,
		env: make(map[string]string),
	}
}

func Run(
	ctx context.Context,
	name string,
	opts ...option,
) (*Result, error) {
	return New(ctx, name).With(opts...).Run()
}

func (c *Cmd) With(opts ...option) *Cmd {
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *Cmd) Run() (*Result, error) {
	c.applyEnv()
	if err := c.cmd.Start(); err != nil {
		return nil, err
	}
	if c.onStarted != nil {
		c.onStarted(c.cmd.Process)
	}
	c.res.exitErr = c.cmd.Wait()
	c.res.processState = c.cmd.ProcessState
	return &c.res, nil
}

func (c *Cmd) applyEnv() {
	if c.envReset {
		c.cmd.Env = make([]string, 0, len(c.env))
		for k, v := range c.env {
			c.cmd.Env = append(c.cmd.Env, k+"="+v)
		}
	} else if len(c.env) > 0 {
		curEnv := os.Environ()
		fullEnv := make(map[string]string, len(curEnv)+len(c.env))
		for _, e := range curEnv {
			name, val, _ := strings.Cut(e, "=")
			fullEnv[name] = val
		}
		maps.Copy(fullEnv, c.env)
		c.cmd.Env = make([]string, 0, len(fullEnv))
		for k, v := range fullEnv {
			c.cmd.Env = append(c.cmd.Env, k+"="+v)
		}
	}
}
