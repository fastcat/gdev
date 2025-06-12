package nodejs

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"fastcat.org/go/gdev/addons/build"
	"fastcat.org/go/gdev/shx"
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
func (n *npmBuilder) BuildAll(ctx context.Context, opts build.Options) error {
	shOpts := []shx.Option{shx.WithCwd(n.root)}
	shOpts = append(shOpts, opts.ShellOpts()...)
	res, err := shx.Run(ctx, []string{"npm", "run", n.buildScript}, shOpts...)
	if err != nil {
		return fmt.Errorf("failed to start npm run %s: %w", n.buildScript, err)
	}
	defer res.Close() // nolint:errcheck
	if err = res.Err(); err != nil {
		if !opts.Verbose {
			_, _ = io.Copy(os.Stderr, res.Stdout())
		}
		return fmt.Errorf("npm run %s failed: %w", n.buildScript, err)
	}
	if err := res.Close(); err != nil {
		return fmt.Errorf("error cleaning up after npm run %s: %w", n.buildScript, err)
	}
	return nil
}

// BuildDirs implements build.Builder.
//
// There is no subdir support for npm, so this just calls BuildAll.
func (n *npmBuilder) BuildDirs(ctx context.Context, _ []string, opts build.Options) error {
	// no subdirs for npm, just build the root
	return n.BuildAll(ctx, opts)
}

// ValidSubdirs implements build.Builder.
//
// There is no subdir support for npm, so this returns nil.
func (n *npmBuilder) ValidSubdirs(ctx context.Context) ([]string, error) {
	// no subdir support for npm
	return nil, nil
}
