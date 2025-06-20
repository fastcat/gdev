package golang

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"

	"fastcat.org/go/gdev/addons/build"
	"fastcat.org/go/gdev/shx"
)

func detectMage(root string) (build.Builder, error) {
	if _, err := os.Stat(filepath.Join(root, "magefile.go")); err == nil {
		return &mageBuilder{root: root}, nil
	} else if _, err := os.Stat(filepath.Join(root, "magefiles")); err == nil {
		return &mageBuilder{root: root}, nil
	}
	return nil, nil // no magefile.go or magefiles directory, not a Mage project
}

type mageBuilder struct {
	root    string
	mageCmd []string
}

// Root implements build.Builder.
func (b *mageBuilder) Root() string {
	return b.root
}

func (b *mageBuilder) resolveMageCmd() []string {
	if len(b.mageCmd) > 0 {
		return b.mageCmd
	}
	// see if mage is in PATH
	if p, err := exec.LookPath("mage"); err == nil {
		b.mageCmd = []string{p}
	} else {
		// assume it's setup with go tool
		b.mageCmd = []string{"go", "tool", "mage"}
	}
	return b.mageCmd
}

// BuildAll implements build.Builder.
func (b *mageBuilder) BuildAll(ctx context.Context, opts build.Options) error {
	shOpts := []shx.Option{shx.WithCwd(b.root)}
	shOpts = append(shOpts, opts.ShellOpts()...)
	cna := append([]string{}, b.resolveMageCmd()...)
	cna = append(cna, "-v")
	res, err := shx.Run(ctx, cna, shOpts...)
	return buildResult("mage", res, err)
}

// BuildDirs implements build.Builder.
//
// Mage does not support building subdirs, so this just calls BuildAll.
func (b *mageBuilder) BuildDirs(ctx context.Context, _ []string, opts build.Options) error {
	return b.BuildAll(ctx, opts)
}

// ValidSubdirs implements build.Builder.
//
// MAge does not support building subdirs
func (b *mageBuilder) ValidSubdirs(context.Context) ([]string, error) {
	return nil, nil
}
