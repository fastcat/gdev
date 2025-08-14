package github

import (
	"fmt"
	"io"
	"strings"

	"fastcat.org/go/gdev/addons/bootstrap"
	"fastcat.org/go/gdev/addons/bootstrap/apt"
	"fastcat.org/go/gdev/shx"
)

type GHLoginOpts struct {
	// value for `--git-protocol` arg, "https" or "ssh", or empty string to omit
	// the arg.
	GitProtocol string
	// value for `--hostname` arg, or empty string to omit it
	Hostname string
	// values to pass to `--scopes`, or nil/empty to omit it
	Scopes []string
}

func GHLoginStep(opts GHLoginOpts) *bootstrap.Step {
	checkCmd := []string{"gh", "auth", "status", "--active"}
	loginCmd := []string{"gh", "auth", "login"}
	if opts.GitProtocol != "" {
		loginCmd = append(loginCmd, "--git-protocol", opts.GitProtocol)
	}
	if opts.Hostname != "" {
		checkCmd = append(checkCmd, "--hostname", opts.Hostname)
		loginCmd = append(loginCmd, "--hostname", opts.Hostname)
	}
	if len(opts.Scopes) > 0 {
		loginCmd = append(loginCmd, "--scopes", strings.Join(opts.Scopes, ","))
	}
	return bootstrap.NewStep(
		AuthLoginStepName,
		func(ctx *bootstrap.Context) error {
			// check if it's already logged in
			if res, err := shx.Run(ctx, checkCmd, shx.CaptureCombined()); err != nil {
				return err
			} else {
				defer res.Close() //nolint:errcheck
				// exit error means not logged in
				if res.Err() == nil {
					// verify expected confirmation appears in output
					if out, err := io.ReadAll(res.Stdout()); err != nil {
						// wtf
						return err
					} else if strings.Contains(string(out), "Logged in to") {
						// ok!
						fmt.Println("Skip: gh already logged in")
						return nil
					}
				}
			}
			if res, err := shx.Run(
				ctx,
				loginCmd,
				shx.PassStdio(),
				shx.WithCombinedError(),
			); err != nil {
				return err
			} else if err := res.Close(); err != nil {
				return err
			}
			return nil
		},
		bootstrap.AfterSteps(apt.StepNameInstall),
	)
}

const AuthLoginStepName = "gh auth login"
