package input

import (
	"context"
	"io"
	"os/user"
	"strings"

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

// NameFromGitConfig reads the user's name from their global git config.
//
// This is generally trustworthy as a loader.
func NameFromGitConfig() Provider[string] {
	return func(ctx context.Context) (string, bool, error) {
		res, err := shx.Run(ctx, []string{"git", "config", "--global", "--includes", "user.name"},
			shx.CaptureOutput(),
		)
		if err != nil {
			return "", false, err
		}
		defer res.Close() // nolint:errcheck
		if res.Err() != nil {
			// git returned an error, just means it doesn't have this config set generally, ignore this
			return "", false, nil
		}
		nameBytes, err := io.ReadAll(res.Stdout())
		if err != nil {
			return "", false, err
		}
		name := strings.TrimSpace(string(nameBytes))
		return name, name != "", nil
	}
}

// WriteNameToGitConfig returns a Writer that writes the given name to the
// user's global (home directory) git config under `user.name`.
func WriteNameToGitConfig() Writer[string] {
	return func(ctx context.Context, name string) error {
		if name == "" {
			// don't write am empty name
			// TODO: return an error here?
			return nil
		}
		if _, err := shx.Run(ctx, []string{"git", "config", "set", "--global", "user.name", name},
			shx.PassOutput(),
		); err != nil {
			return err
		}
		return nil
	}
}

var UserNameKey = internal.NewKey[string]("user name")

func UserNamePrompt() *Prompter[string] {
	return TextPrompt(
		UserNameKey,
		"What's your full name?",
		WithLoaders(NameFromGitConfig()),
		WithGuessers(NameFromPasswd()),
		WithWriters(WriteNameToGitConfig()),
	)
}
