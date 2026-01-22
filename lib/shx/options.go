package shx

import (
	"fmt"
	"io"
	"maps"
	"os"
	"os/exec"
	"slices"
	"strings"
	"sync"

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

func CaptureError() Option {
	return optionExecFunc(func(cmd *exec.Cmd, res *Result) {
		if res.stderrCapture != nil {
			_ = res.stderrCapture.Close()
		}
		res.stderrCapture = &outCapture{}
		res.stderrCapture.init()
		cmd.Stderr = res.stderrCapture
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

var usingSudoRS = sync.OnceValue(func() bool {
	// sudo-rs formats auth prompts differently, adjust how we set SUDO_PROMPT
	// based on whether we are using "traditional" sudo (aka sudo.ws) or sudo-rs.
	out, err := exec.Command("sudo", "--version").CombinedOutput()
	if err != nil {
		// assume traditional sudo if we can't determine otherwise
		return false
	}
	return strings.HasPrefix(string(out), "sudo-rs ")
})

func WithSudo(purpose string) Option {
	return WithSudoUser("", purpose)
}

func WithSudoUser(user, purpose string) Option {
	var shxCmd *Cmd
	return optionFuncs{
		cmd: func(c *Cmd) {
			// -u and -E args will get inserted later
			c.cmdAndArgs = append([]string{"sudo"}, c.cmdAndArgs...)
			shxCmd = c
		},
		exec: func(cmd *exec.Cmd, _ *Result) {
			// insert options after sudo
			var newOpts []string
			if user != "" {
				newOpts = append(newOpts, "-u", user)
			}
			// any WithEnv needs to get passed through sudo. depending on which sudo
			// we're using, there's different ways. prefer the more secure way that
			// "original" sudo supports if we can, else the insecure way that sudo-rs
			// requires if we can't.
			if usingSudoRS() {
				// sudo-rs doesn't support a list of env names to pass through, requires
				// the values to be exposed on the CLI where they are visible to any
				// process/user on the system
				for k, v := range shxCmd.env {
					newOpts = append(newOpts, fmt.Sprintf("%s=%s", k, v))
				}
			} else {
				if len(shxCmd.env) > 0 {
					newOpts = append(newOpts,
						"--preserve-env="+strings.Join(slices.Collect(maps.Keys(shxCmd.env)), ","),
					)
				}
			}
			if len(newOpts) > 0 {
				newArgs := make([]string, 0, len(cmd.Args)+len(newOpts))
				newArgs = append(newArgs, cmd.Args[0])
				newArgs = append(newArgs, newOpts...)
				newArgs = append(newArgs, cmd.Args[1:]...)
				cmd.Args = newArgs
			}
			cmd.Env = append(cmd.Env, "SUDO_ASKPASS=1")
			if usingSudoRS() {
				cmd.Env = append(cmd.Env, fmt.Sprintf(
					"SUDO_PROMPT=%s needs the password for %%p to %s",
					internal.AppName(),
					purpose,
				))
			} else {
				cmd.Env = append(cmd.Env, fmt.Sprintf(
					"SUDO_PROMPT=%s needs the password for %%p to %s: ",
					internal.AppName(),
					purpose,
				))
			}
		},
	}
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
