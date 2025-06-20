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

func detectNPM(root string) (build.Builder, error) {
	pjPath := filepath.Join(root, "package.json")
	if _, err := os.Stat(pjPath); err != nil {
		return nil, nil // no package.json, not an npm project
	}
	pj, err := internal.ReadJSONFile[PackageJSON](pjPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", pjPath, err)
	}
	b := &npmBuilder{
		tool:        "npm",
		root:        root,
		buildScript: "build",
		pj:          pj,
	}
	// if there are workspaces, expand them out
	if b.subdirs, err = expandWorkspaces(root, pj.Workspaces); err != nil {
		return nil, err
	}

	// detect if this is really a pnpm project
	if _, err := os.Stat(filepath.Join(root, "pnpm-lock.yaml")); err == nil {
		// it's a pnpm project, adjust
		b.tool = "pnpm"
		if len(pj.Workspaces) != 0 {
			// have to use pnpm workspace if using pnpm
			return nil, fmt.Errorf("cannot use npm workspaces with pnpm in %s", pjPath)
		}
		wsPath := filepath.Join(root, "pnpm-workspace.yaml")
		b.ws, err = internal.ReadJSONFile[*PNPMWorkspaceYAML](wsPath)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("failed to read %s: %w", wsPath, err)
		} else if b.ws != nil {
			// if there are workspaces, expand them out
			if b.subdirs, err = expandWorkspaces(root, b.ws.Packages); err != nil {
				return nil, err
			}
		}

	}

	// TODO: check if there's a `build` script in package.json
	return b, nil
}

type npmBuilder struct {
	tool        string // "npm", "pnpm", ...
	root        string
	buildScript string
	pj          PackageJSON
	ws          *PNPMWorkspaceYAML // may not exist, only valid if tool is pnpm
	subdirs     []string
}

func (b *npmBuilder) build(
	ctx context.Context,
	args []string,
	opts build.Options,
) error {
	shOpts := []shx.Option{shx.WithCwd(b.root)}
	shOpts = append(shOpts, opts.ShellOpts()...)
	cna := []string{b.tool, "run"}
	cna = append(cna, args...)
	cna = append(cna, b.buildScript)
	res, err := shx.Run(ctx, cna, shOpts...)
	if err != nil {
		return fmt.Errorf("failed to start %s run %s: %w", b.tool, b.buildScript, err)
	}
	defer res.Close() // nolint:errcheck
	if err = res.Err(); err != nil {
		if !opts.Verbose {
			_, _ = io.Copy(os.Stderr, res.Stdout())
		}
		return fmt.Errorf("%s run %s failed: %w", b.tool, b.buildScript, err)
	}
	if err := res.Close(); err != nil {
		return fmt.Errorf("error cleaning up after npm run %s: %w", b.buildScript, err)
	}
	return nil
}

// BuildAll implements build.Builder.
func (b *npmBuilder) BuildAll(ctx context.Context, opts build.Options) error {
	return b.build(ctx, nil, opts)
}

// BuildDirs implements build.Builder.
func (b *npmBuilder) BuildDirs(ctx context.Context, dirs []string, opts build.Options) error {
	args := make([]string, 0, len(dirs))
	var argBase string
	switch b.tool {
	case "npm":
		argBase = "--workspace="
	case "pnpm":
		argBase = "--filter="
	default:
		return fmt.Errorf("unsupported tool %s for building npm-ish workspaces", b.tool)
	}

	for _, dir := range dirs {
		// `./` prefix is required for the filter to be understood as a path in
		// pnpm, is OK for npmq
		if !strings.HasPrefix(dir, "./") {
			dir = "./" + dir
		}
		args = append(args, argBase+dir)
	}
	return b.build(ctx, args, opts)
}

// ValidSubdirs implements build.Builder.
func (b *npmBuilder) ValidSubdirs(context.Context) ([]string, error) {
	return b.subdirs, nil
}
