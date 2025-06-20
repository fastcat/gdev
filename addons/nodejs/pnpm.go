package nodejs

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"fastcat.org/go/gdev/addons/build"
	"fastcat.org/go/gdev/internal"
	"fastcat.org/go/gdev/shx"
)

// TODO: should this just share code with npm?

func detectPNPM(root string) (build.Builder, error) {
	pjPath := filepath.Join(root, "package.json")
	if _, err := os.Stat(pjPath); err != nil {
		return nil, nil // no package.json, not a (p)npm project
	}
	if _, err := os.Stat(filepath.Join(root, "pnpm-lock.yaml")); err != nil {
		return nil, nil // no pnpm lockfile, not a pnpm project
	}
	pj, err := internal.ReadJSONFile[PackageJSON](pjPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", pjPath, err)
	}
	// can't use npm workspaces with pnpm, have to use pnpm workspaces or just use npm
	if len(pj.Workspaces) != 0 {
		return nil, fmt.Errorf("cannot use npm workspaces with pnpm in %s", pjPath)
	}
	b := &pnpmBuilder{
		root:        root,
		buildScript: "build",
		pj:          pj,
	}
	wsPath := filepath.Join(root, "pnpm-workspace.yaml")
	b.ws, err = internal.ReadJSONFile[*PNPMWorkspacesYAML](wsPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("failed to read %s: %w", wsPath, err)
	} else if b.ws != nil {
		// if there are workspaces, expand them out
		if b.subdirs, err = expandWorkspaces(root, b.ws.Packages); err != nil {
			return nil, err
		}
	}
	// TODO: check if there's a `build` script in package.json
	return b, nil
}

type pnpmBuilder struct {
	root        string
	buildScript string
	pj          PackageJSON
	ws          *PNPMWorkspacesYAML // may not exist
	subdirs     []string
}

func (n *pnpmBuilder) build(
	ctx context.Context,
	args []string,
	opts build.Options,
) error {
	shOpts := []shx.Option{shx.WithCwd(n.root)}
	shOpts = append(shOpts, opts.ShellOpts()...)
	cna := []string{"pnpm", "run"}
	cna = append(cna, args...)
	cna = append(cna, n.buildScript)
	res, err := shx.Run(ctx, cna, shOpts...)
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

// BuildAll implements build.Builder.
func (n *pnpmBuilder) BuildAll(ctx context.Context, opts build.Options) error {
	return n.build(ctx, nil, opts)
}

// BuildDirs implements build.Builder.
//
// This will include `--filter=...` arg(s) for each subdir.
func (n *pnpmBuilder) BuildDirs(ctx context.Context, dirs []string, opts build.Options) error {
	args := make([]string, 0, len(dirs))
	for _, dir := range dirs {
		// `./` prefix is required for the filter to be understood as a path
		if !strings.HasPrefix(dir, "./") {
			dir = "./" + dir
		}
		args = append(args, "--filter="+dir)
	}
	return n.build(ctx, args, opts)
}

// ValidSubdirs implements build.Builder.
//
// This will return the workspace directories, if any, from the root
// `pnpm-workspace.json`, with globs expanded.
func (n *pnpmBuilder) ValidSubdirs(ctx context.Context) ([]string, error) {
	return n.subdirs, nil
}
