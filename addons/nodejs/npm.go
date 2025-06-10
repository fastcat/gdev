package nodejs

import (
	"os"
	"path/filepath"

	"fastcat.org/go/gdev/addons/build"
)

func detectNPM(root string) (build.Builder, error) {
	if _, err := os.Stat(filepath.Join(root, "package.json")); err != nil {
		return nil, nil // no package.json, not an npm project
	}
	// TODO: check if there's a `build` script in package.json
	return &npmBuilder{
		root:        root,
		buildScript: "build",
	}, nil
}

type npmBuilder struct {
	root        string
	buildScript string
}

// BuildAll implements build.Builder.
func (n *npmBuilder) BuildAll() error {
	panic("unimplemented")
}

// BuildDirs implements build.Builder.
//
// There is no subdir support for npm, so this just calls BuildAll.
func (n *npmBuilder) BuildDirs(dirs []string) error {
	// no subdirs for npm, just build the root
	return n.BuildAll()
}

// ValidSubdirs implements build.Builder.
//
// There is no subdir support for npm, so this returns nil.
func (n *npmBuilder) ValidSubdirs() ([]string, error) {
	// no subdir support for npm
	return nil, nil
}
