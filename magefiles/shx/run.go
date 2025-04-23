package shx

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/magefile/mage/mg"
)

func Run(
	ctx context.Context,
	cmd string,
	args ...string,
) error {
	return Cmd(ctx, cmd, args...).Run()
}

func Cmd(
	ctx context.Context,
	command string,
	args ...string,
) *cmd {
	c := exec.CommandContext(ctx, command, args...)
	c.Env = os.Environ()
	c.Stdout, c.Stderr = nil, os.Stderr
	if mg.Verbose() {
		c.Stdout = os.Stdout
	}
	return (*cmd)(c)
}

type cmd exec.Cmd

func (c *cmd) With(opts ...execOpt) *cmd {
	for _, o := range opts {
		o((*exec.Cmd)(c))
	}
	return c
}

func (c *cmd) Run() error {
	if mg.Verbose() {
		quoted := make([]string, 0, len(c.Args))
		for _, a := range c.Args {
			quoted = append(quoted, strconv.Quote(a))
		}
		log.Println("exec:", c.Path, strings.Join(quoted, " "))
	}
	err := (*exec.Cmd)(c).Run()
	if err != nil {
		name := filepath.Base(c.Path)
		if name == "go" && len(c.Args) > 0 {
			name += " " + c.Args[0]
		}
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}

type execOpt func(*exec.Cmd)

func WithEnv(env map[string]string) execOpt {
	return func(c *exec.Cmd) {
		for k, v := range env {
			c.Env = append(c.Env, k+"="+v)
		}
	}
}

func WithExpandArgs() execOpt {
	return func(c *exec.Cmd) {
		env := make(map[string]string, len(c.Env))
		for _, e := range c.Env {
			k, v, _ := strings.Cut(e, "=")
			env[k] = v
		}
		exp := func(k string) string { return env[k] }
		for i, a := range c.Args {
			c.Args[i] = os.Expand(a, exp)
		}
	}
}

// WithOutput makes the command always pass stdout even if not invoked with
// `mage -v`.
func WithOutput() execOpt {
	return func(c *exec.Cmd) {
		c.Stdout = os.Stdout
	}
}
