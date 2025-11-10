package shx

import (
	"fmt"
	"io"
	"os"
	"os/exec"

	"fastcat.org/go/gdev/internal"
)

type Option interface {
	apply(*Cmd)
	applyExec(*exec.Cmd, *Result)
}

type optionCmdFunc func(*Cmd)

func (f optionCmdFunc) apply(c *Cmd) {
	f(c)
}
func (f optionCmdFunc) applyExec(cmd *exec.Cmd, res *Result) {}

type optionExecFunc func(cmd *exec.Cmd, res *Result)

func (f optionExecFunc) apply(c *Cmd) {}
func (f optionExecFunc) applyExec(cmd *exec.Cmd, res *Result) {
	f(cmd, res)
}

/*
type optionFuncs struct {
	cmd  optionCmdFunc
	exec optionExecFunc
}

func (f optionFuncs) apply(c *Cmd) {
	if f.cmd != nil {
		f.cmd(c)
	}
}

func (f optionFuncs) applyExec(cmd *exec.Cmd, res *Result) {
	if f.exec != nil {
		f.exec(cmd, res)
	}
}
*/

// WithCombinedError changes the behavior of Run to return all errors in the
// error return, instead of only returning errors starting the process there,
// and errors from the process in the Result.
func WithCombinedError() Option {
	return optionCmdFunc(func(c *Cmd) {
		c.combineExecErrors = true
	})
}

func WithCwd(path string) Option {
	return optionExecFunc(func(c *exec.Cmd, r *Result) {
		c.Dir = path
	})
}

func CaptureOutput() Option {
	return optionExecFunc(func(cmd *exec.Cmd, res *Result) {
		if res.stdoutCapture != nil {
			_ = res.stdoutCapture.Close()
		}
		res.stdoutCapture = &outCapture{}
		res.stdoutCapture.init()
		cmd.Stdout = res.stdoutCapture
	})
}

func CaptureCombined() Option {
	return optionExecFunc(func(cmd *exec.Cmd, res *Result) {
		if res.stdoutCapture != nil {
			_ = res.stdoutCapture.Close()
		}
		if res.stderrCapture != nil {
			_ = res.stderrCapture.Close()
		}
		res.stdoutCapture = &outCapture{}
		res.stdoutCapture.init()
		res.stderrCapture = res.stdoutCapture
		cmd.Stdout = res.stdoutCapture
		cmd.Stderr = res.stdoutCapture
	})
}

// PassStdout sets the command's Stdout to os.Stdout and clears any prior
// capture configuration.
func PassStdout() Option {
	return optionExecFunc(func(cmd *exec.Cmd, res *Result) {
		if res.stdoutCapture != nil {
			_ = res.stdoutCapture.Close()
			res.stdoutCapture = nil
		}
		cmd.Stdout = os.Stdout
	})
}

// PassStderr sets the command's Stderr to os.Stderr and clears any prior
// capture configuration.
func PassStderr() Option {
	return optionExecFunc(func(cmd *exec.Cmd, res *Result) {
		if res.stderrCapture != nil {
			_ = res.stderrCapture.Close()
			res.stderrCapture = nil
		}
		cmd.Stderr = os.Stderr
	})
}

// PassStdin sets the command's Stdin to os.Stdin.
func PassStdin() Option {
	return optionExecFunc(func(cmd *exec.Cmd, res *Result) {
		cmd.Stdin = os.Stdin
	})
}

// PassOutput sets the command's Stdout and Stderr to os.Stdout and os.Stderr
// respectively, and clears any prior capture configuration.
func PassOutput() Option {
	return optionExecFunc(func(cmd *exec.Cmd, res *Result) {
		if res.stdoutCapture != nil {
			_ = res.stdoutCapture.Close()
			res.stdoutCapture = nil
		}
		if res.stderrCapture != nil {
			_ = res.stderrCapture.Close()
			res.stderrCapture = nil
		}
		cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	})
}

func PassStdio() Option {
	return optionExecFunc(func(cmd *exec.Cmd, res *Result) {
		if res.stdoutCapture != nil {
			_ = res.stdoutCapture.Close()
			res.stdoutCapture = nil
		}
		if res.stderrCapture != nil {
			_ = res.stderrCapture.Close()
			res.stderrCapture = nil
		}
		cmd.Stdout, cmd.Stderr, cmd.Stdin = os.Stdout, os.Stderr, os.Stdin
	})
}

// FeedStdin sets the command's Stdin to the provided io.Reader.
//
// The caller is responsible for closing the reader if necessary after the
// command completes.
func FeedStdin(in io.Reader) Option {
	return optionExecFunc(func(cmd *exec.Cmd, res *Result) {
		cmd.Stdin = in
	})
}

func WithSudo(purpose string) Option {
	return optionCmdFunc(func(c *Cmd) {
		c.cmdAndArgs = append([]string{"sudo"}, c.cmdAndArgs...)
		c.env["SUDO_ASKPASS"] = "1"
		c.env["SUDO_PROMPT"] = fmt.Sprintf(
			"%s needs the password for %%p to %s: ",
			internal.AppName(),
			purpose,
		)
	})
}

func WithSudoUser(user, purpose string) Option {
	return optionCmdFunc(func(c *Cmd) {
		c.cmdAndArgs = append([]string{"sudo", "-u", user}, c.cmdAndArgs...)
		c.env["SUDO_ASKPASS"] = "1"
		c.env["SUDO_PROMPT"] = fmt.Sprintf(
			"%s needs the password for %%p to %s: ",
			internal.AppName(),
			purpose,
		)
	})
}

func WithUmask(umask os.FileMode) Option {
	// changing the umask requires hacks, see
	// https://github.com/golang/go/issues/56016. those hacks often don't work in
	// e.g. containers or other constrained environments (unshare FS => operation
	// not permitted), so use /bin/sh hacks instead
	return optionCmdFunc(func(c *Cmd) {
		c.cmdAndArgs = append(
			[]string{
				"/bin/sh", "-c",
				fmt.Sprintf("umask 0%03o && exec \"$@\"", umask),
				// this allows us to pass the original command and args as positional
				// parameters without having to worry about any shell quoting/escaping
				// rules
				"--",
			},
			c.cmdAndArgs...,
		)
	})
}

func WithEnv(key, value string) Option {
	return optionCmdFunc(func(c *Cmd) {
		c.env[key] = value
	})
}
