package input

import (
	"context"
	"fmt"
	"io"
	"os/user"
	"slices"
	"strings"

	"fastcat.org/go/gdev/addons/bootstrap"
	"fastcat.org/go/gdev/addons/bootstrap/apt"
	"fastcat.org/go/gdev/addons/bootstrap/internal"
	"fastcat.org/go/gdev/lib/shx"
)

// NameFromOS guesses the user's name from the system user info.
//
// This should not be trusted to use as a Loader, since container and other
// environments often have nonsense here.
func NameFromOS() Provider[string] {
	return func(ctx context.Context) (string, bool, error) {
		u, err := user.Current()
		if err != nil {
			return "", false, err
		}
		return u.Name, true, nil
	}
}

func readGitConfigString(ctx context.Context, name string) (string, error) {
	// use the legacy `--get` option to avoid any possible ambiguity
	res, err := shx.Run(ctx, []string{
		"git",
		"config",
		"--global",
		"--includes",
		"--get",
		name,
	},
		shx.CaptureOutput(),
	)
	if err != nil {
		return "", err
	}
	defer res.Close() // nolint:errcheck
	if res.Err() != nil {
		// git returned an error, just means it doesn't have this config set generally, ignore this
		return "", nil
	}
	valueBytes, err := io.ReadAll(res.Stdout())
	if err != nil {
		return "", err
	}
	value := strings.TrimSpace(string(valueBytes))
	return value, nil
}

// ReadGitConfigString returns a Provider that reads a string value from the
// user's global git config.
//
// Common names you might want to read include:
//
//   - `user.name`
//   - `user.email`
//   - `github.user`
//
// These are generally trustworthy as a loader.
func ReadGitConfigString(name string) Provider[string] {
	return func(ctx context.Context) (string, bool, error) {
		value, err := readGitConfigString(ctx, name)
		return value, value != "", err
	}
}

// WriteGitConfigString returns a Writer that writes a value for the given name to the
// user's global (home directory) git config
func WriteGitConfigString(name string) Writer[string] {
	return func(ctx context.Context, value string) error {
		if value == "" {
			// don't write am empty value
			// TODO: return an error here?
			return nil
		}
		oldVal, err := readGitConfigString(ctx, name)
		if err == nil && oldVal == value {
			// no change, don't write
			return nil
		}
		// modern git supports `config set --global name value`, but older versions
		// are still common, e.g. Ubuntu 24.04, so use the legacy syntax that omits
		// the `set` verb.
		if _, err := shx.Run(ctx, []string{"git", "config", "--global", name, value},
			shx.PassOutput(),
			shx.WithCombinedError(),
		); err != nil {
			return err
		}
		return nil
	}
}

var (
	UserNameKey   = internal.NewKey[string]("user name")
	GitHubUserKey = internal.NewKey[string]("github user")
)

// UserNamePrompt prompts for the `user.name` git setting, guessing it from the
// OS username.
func UserNamePrompt() *Prompter[string] {
	return TextPrompt(
		UserNameKey,
		"What's your full name?",
		WithLoaders(ReadGitConfigString("user.name")),
		WithGuessers(NameFromOS()),
		WithWriters(WriteGitConfigString("user.name")),
	)
}

// UserEmailPrompt prompts for the `user.email` git setting.
//
// It will use the guesser from [SetUserEmailGuesser], if any.
//
// TODO: allow providing a custom guesser for this, since most companies will
// have predictable emails from the person's name.
func UserEmailPrompt() *Prompter[string] {
	return TextPrompt(
		internal.NewKey[string]("user email"),
		"What's your email address (for git)?",
		WithLoaders(ReadGitConfigString("user.email")),
		WithGuessers(func(ctx context.Context) (string, bool, error) {
			if userEmailGuesser != nil {
				fullName, ok, err := ReadGitConfigString("user.name")(ctx)
				if err != nil {
					return "", false, err
				}
				if !ok || fullName == "" {
					// try again with other guesser
					if fullName, ok, err = NameFromOS()(ctx); err != nil {
						return "", false, err
					}
				}
				if ok && fullName != "" {
					email := userEmailGuesser(fullName)
					if email != "" {
						return email, true, nil
					}
				}
			}
			return "", false, nil
		}),
		WithValidator(func(email string) error {
			// TODO: use "the" email regexp / parser rules instead of this simplistic stuff
			local, domain, ok := strings.Cut(email, "@")
			if !ok || local == "" || domain == "" {
				return fmt.Errorf("invalid email format")
			}
			if len(userEmailDomains) > 0 && !slices.Contains(userEmailDomains, domain) {
				return fmt.Errorf("email domain %q is not allowed", domain)
			}
			return nil
		}),
		WithWriters(WriteGitConfigString("user.email")),
	)
}

var (
	userEmailGuesser func(fullName string) (email string)
	userEmailDomains []string
)

// TODO: this should be some kind of bootstrap config option
func SetUserEmailGuesser(guesser func(fullName string) (email string)) {
	userEmailGuesser = guesser
}

// Require the user email entered to be in one of the listed domains. Do not
// include the `@`.
func RequireUserEmailDomains(validDomains ...string) {
	userEmailDomains = slices.Clone(validDomains)
}

// GitHubUserPrompt prompts for the `github.user` git setting, with no guessing.
func GitHubUserPrompt() *Prompter[string] {
	return TextPrompt(
		GitHubUserKey,
		"What's your GitHub username?",
		WithLoaders(ReadGitConfigString("github.user")),
		// no good guessing rule here sadly
		WithWriters(WriteGitConfigString("github.user")),
	)
}

const StepNameUserInfo = "Get user info"

// UserInfoStep returns a bootstrap step that prompts for the user's name,
// email, and GitHub username.
//
// TODO: add a way to customize the email guessing, see [UserEmailPrompt].
func UserInfoStep() *bootstrap.Step {
	return PromptStep(
		StepNameUserInfo,
		UserNamePrompt(),
		UserEmailPrompt(),
		GitHubUserPrompt(),
	).With(
		bootstrap.AfterSteps(apt.StepNameInstall),
	)
}
