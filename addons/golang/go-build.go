package golang

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"fastcat.org/go/gdev/addons/build"
	"fastcat.org/go/gdev/shx"
)

func detectGoBuild(root string) (build.Builder, error) {
	// TODO: recognize go.work without a go.mod in the root
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		return nil, nil // no package.json, not an npm project
	}
	// TODO: check if there's a `build` script in package.json
	return &goBuilder{
		root: root,
		// workspace: false
	}, nil
}

type goBuilder struct {
	root      string
	workspace bool
}

// BuildAll implements build.Builder.
func (b *goBuilder) BuildAll(ctx context.Context, opts build.Options) error {
	if b.workspace {
		return fmt.Errorf("not implemented: go workspace build")
	}
	shOpts := []shx.Option{shx.WithCwd(b.root)}
	shOpts = append(shOpts, opts.ShellOpts()...)
	res, err := shx.Run(ctx,
		[]string{"go", "build", "-v", "./..."},
		shOpts...,
	)
	return buildResult("go build", res, err)
}

// BuildDirs implements build.Builder.
func (b *goBuilder) BuildDirs(ctx context.Context, dirs []string, opts build.Options) error {
	if len(dirs) == 0 {
		return b.BuildAll(ctx, opts)
	}
	cna := []string{"go", "build", "-v"}
	for _, d := range dirs {
		cna = append(cna, filepath.Join(b.root, "./"+filepath.Clean(d)+"/..."))
	}
	shOpts := []shx.Option{shx.WithCwd(b.root)}
	shOpts = append(shOpts, opts.ShellOpts()...)
	res, err := shx.Run(ctx, cna, shOpts...)
	return buildResult("go build", res, err)
}

// ValidSubdirs implements build.Builder.
//
// TODO: detect modules in the workspace
func (b *goBuilder) ValidSubdirs(ctx context.Context) ([]string, error) {
	return nil, nil
}
