package nodejs

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"fastcat.org/go/gdev/addons/build"
	"fastcat.org/go/gdev/internal"
	"fastcat.org/go/gdev/lib/shx"
)

func detectRush(root string) (build.Builder, error) {
	rjPath := filepath.Join(root, "rush.json")
	if _, err := os.Stat(rjPath); err != nil {
		return nil, nil // no rush.json, not a Rush project
	}
	rj, err := internal.ReadJSONFile[RushJSON](rjPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", rjPath, err)
	}
	return &rushBuilder{
		root: root,
		rj:   rj,
	}, nil
}

// Root implements build.Builder.
func (b *rushBuilder) Root() string {
	return b.root
}

type rushBuilder struct {
	root string
	rj   RushJSON

	// populated on demand
	dirToPkg map[string]string
}

func (b *rushBuilder) withExtra() {
	if b.dirToPkg == nil {
		b.dirToPkg = make(map[string]string, len(b.rj.Projects))
		for _, p := range b.rj.Projects {
			b.dirToPkg[p.ProjectFolder] = p.PackageName
		}
	}
}

func (b *rushBuilder) build(
	ctx context.Context,
	args []string,
	opts build.Options,
) error {
	shOpts := []shx.Option{shx.WithCwd(b.root)}
	shOpts = append(shOpts, opts.ShellOpts()...)
	// always tell rush to emit verbose output so we can emit it on errors, rush's
	// tendency to only emit the tail of a build error often doesn't include
	// enough context
	cna := []string{"rush", "build", "--verbose"}
	cna = append(cna, args...)
	res, err := shx.Run(ctx, cna, shOpts...)
	if err != nil {
		return fmt.Errorf("failed to start rush build: %w", err)
	}
	defer res.Close() //nolint:errcheck
	if err = res.Err(); err != nil {
		if !opts.Verbose {
			_, _ = io.Copy(os.Stderr, res.Stdout())
		}
		return fmt.Errorf("rush build failed: %w", err)
	}
	if err := res.Close(); err != nil {
		return fmt.Errorf("error cleaning up after rush build: %w", err)
	}
	return nil
}

// BuildAll implements build.Builder.
func (b *rushBuilder) BuildAll(ctx context.Context, opts build.Options) error {
	return b.build(ctx, nil, opts)
}

// BuildDirs implements build.Builder.
func (b *rushBuilder) BuildDirs(ctx context.Context, dirs []string, opts build.Options) error {
	args := make([]string, 0, len(dirs)*2)
	b.withExtra()
	for _, dir := range dirs {
		if pkg, ok := b.dirToPkg[dir]; ok {
			args = append(args, "--to", pkg)
		} else {
			return fmt.Errorf("unknown directory %s to build for %s", dir, b.root)
		}
	}
	return b.build(ctx, args, opts)
}

// ValidSubdirs implements build.Builder.
func (b *rushBuilder) ValidSubdirs(context.Context) ([]string, error) {
	ret := make([]string, 0, len(b.rj.Projects))
	for _, p := range b.rj.Projects {
		ret = append(ret, p.ProjectFolder)
	}
	return ret, nil
}
