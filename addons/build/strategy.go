package build

import (
	"context"
	"fmt"

	"fastcat.org/go/gdev/lib/shx"
)

// Builder represents a tool that can build a repo, or a set of subdirs within
// the repo.
type Builder interface {
	// Root gives the root directory the builder targets.
	Root() string
	// BuildAll builds the whole repo
	BuildAll(context.Context, Options) error
	// ValidateSubdirs checks which subdirs are valid to pass to BuildDirs. It
	// should return _relative_ paths.
	ValidSubdirs(context.Context) ([]string, error)
	// BuildDirs builds the specified subdirs. It should expect relative paths,
	// tolerating both presence and absence of a leading `./` or the
	// platform-specific equivalent. It may return an error if any of the dirs is
	// not in what ValidSubdirs would return. It may also do a sensible build if
	// it can infer a sensible one, e.g. building the nearest parent dir.
	BuildDirs(ctx context.Context, dirs []string, opts Options) error
}

type Detector func(root string) (Builder, error)

type Options struct {
	Verbose bool
}

func (o Options) ShellOpts() []shx.Option {
	var opts []shx.Option
	if o.Verbose {
		opts = append(opts, shx.PassOutput())
	} else {
		// caller should copy the builder output to our output if it fails
		opts = append(opts, shx.CaptureCombined())
	}
	return opts
}

type strategy struct {
	name       string
	detector   Detector
	supersedes []string
}

func DetectStrategy(root string) (string, Builder, error) {
	for _, name := range addon.Config.strategyOrder {
		s := addon.Config.strategies[name]
		builder, err := s.detector(root)
		if err != nil {
			return "", nil, fmt.Errorf("error running detector for %s: %w", s.name, err)
		}
		if builder != nil {
			return name, builder, nil
		}
	}
	return "", nil, nil
}

func TryStrategy(root, strategy string) (Builder, error) {
	if s, ok := addon.Config.strategies[strategy]; ok {
		return s.detector(root)
	}
	return nil, fmt.Errorf("unknown strategy %q", strategy)
}
