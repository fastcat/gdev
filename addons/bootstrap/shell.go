package bootstrap

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"

	"fastcat.org/go/gdev/instance"
)

func Shell(
	ctx *Context,
	cmdAndArgs []string,
	opts ...shOpt,
) error {
	for _, o := range opts {
		if o.cli != nil {
			o.cli(&cmdAndArgs)
		}
	}
	cmd := exec.CommandContext(ctx, cmdAndArgs[0], cmdAndArgs[1:]...)
	for _, o := range opts {
		if o.cmd != nil {
			o.cmd(cmd)
		}
	}
	var out []byte
	var err error
	if cmd.Stdout == nil && cmd.Stderr == nil {
		out, err = cmd.CombinedOutput()
	} else if cmd.Stdout == nil {
		out, err = cmd.Output()
	} else if cmd.Stderr == nil {
		panic("unimplemented")
	} else {
		err = cmd.Run()
	}
	if err != nil {
		// copy error output to stderr to help human
		_, _ = io.Copy(os.Stderr, bytes.NewReader(out))
		return err
	}
	return nil
}

type shOpt struct {
	cli func(cmdAndArgs *[]string)
	cmd func(cmd *exec.Cmd)
}

func WithSudo(purpose string) shOpt {
	if os.Geteuid() == 0 {
		// already root, don't actually need sudo, no-op
		return shOpt{}
	}
	return shOpt{
		cli: func(cmdAndArgs *[]string) {
			*cmdAndArgs = append([]string{"sudo"}, *cmdAndArgs...)
		},
		cmd: func(cmd *exec.Cmd) {
			// don't need to mess with stdio, as sudo is very smart about finding the
			// controlling tty and opening that on its own to do prompting
			if cmd.Env == nil {
				cmd.Env = os.Environ()
			}
			cmd.Env = append(cmd.Env, fmt.Sprintf(
				"SUDO_PROMPT=%s needs the password for %%p to %s: ",
				instance.AppName(),
				purpose,
			))
		},
	}
}

func WithPassStdio() shOpt {
	return shOpt{
		cmd: func(cmd *exec.Cmd) {
			cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
		},
	}
}
