package input

import (
	"context"
	"io"
	"os/user"
	"strings"

	"fastcat.org/go/gdev/addons/bootstrap"
	"fastcat.org/go/gdev/addons/bootstrap/internal"
	"fastcat.org/go/gdev/shx"
)

// NameFromPasswd guesses the user's name from the system's passwd file.
//
// This should not be trusted to use as a Loader, since container and other
// environments often have nonsense here.
func NameFromPasswd() Provider[string] {
	return func(ctx context.Context) (string, bool, error) {
		u, err := user.Current()
		if err != nil {
			return "", false, err
		}
		return u.Name, true, nil
	}
}

func readGitConfigString(ctx context.Context, name string) (string, error) {
	res, err := shx.Run(ctx, []string{"git", "config", "--global", "--includes", name},
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
		if _, err := shx.Run(ctx, []string{"git", "config", "set", "--global", name, value},
			shx.PassOutput(),
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

func UserNamePrompt() *Prompter[string] {
	return TextPrompt(
		UserNameKey,
		"What's your full name?",
		WithLoaders(ReadGitConfigString("user.name")),
		WithGuessers(NameFromPasswd()),
		WithWriters(WriteGitConfigString("user.name")),
	)
}

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

func UserInfoStep() *bootstrap.Step {
	return PromptStep(
		StepNameUserInfo,
		UserNamePrompt(),
		GitHubUserPrompt(),
	).With(
		bootstrap.AfterSteps(bootstrap.StepNameAptInstall),
	)
}
