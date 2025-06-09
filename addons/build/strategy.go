package build

import "fmt"

// Builder represents a tool that can build a repo, or a set of subdirs within
// the repo.
type Builder interface {
	// BuildAll builds the whole repo
	BuildAll() error
	// ValidateSubdirs checks which subdirs are valid to pass to BuildDirs. It
	// should return _relative_ paths.
	ValidSubdirs() ([]string, error)
	// BuildDirs builds the specified subdirs. It should expect relative paths,
	// tolerating both presence and absence of a leading `./` or the
	// platform-specific equivalent. It may return an error if any of the dirs is
	// not in what ValidSubdirs would return. It may also do a sensible build if
	// it can infer a sensible one, e.g. building the nearest parent dir.
	BuildDirs(dirs []string) error
}

type Detector func(root string) (Builder, error)

type strategy struct {
	name       string
	detector   Detector
	supersedes []string
}

func DetectStrategy(root string) (Builder, error) {
	for _, name := range addon.Config.strategyOrder {
		s := addon.Config.strategies[name]
		builder, err := s.detector(root)
		if err != nil {
			return nil, fmt.Errorf("error running detector for %s: %w", s.name, err)
		}
		if builder != nil {
			return builder, nil
		}
	}
	return nil, nil
}
