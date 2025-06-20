package build

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"

	"fastcat.org/go/gdev/service"
)

// Services builds the given services, aggregating them by repo root to
// efficiently build subdirs.
func Services(
	ctx context.Context,
	svcs []service.ServiceWithSource,
	opts Options,
) error {
	// TODO: support concurrent builds
	// TODO: support progress bars

	root2idx := map[string]int{}
	var builders []Builder
	var strategies []string
	var subdirs [][]string

	for _, svc := range svcs {
		root, subdir, err := svc.LocalSource(ctx)
		if err != nil {
			return fmt.Errorf("error getting local source for service %s: %w", svc.Name(), err)
		}
		root = filepath.Clean(root)
		if idx, ok := root2idx[root]; ok {
			// append the subdir
			subdirs[idx] = append(subdirs[idx], filepath.Clean(subdir))
			continue
		}

		sn, b, err := DetectStrategy(root)
		if err != nil {
			return fmt.Errorf("error detecting build strategy for %q: %w", root, err)
		} else if b == nil {
			return fmt.Errorf("no build strategy found for %q", root)
		}
		builders = append(builders, b)
		strategies = append(strategies, sn)
		subdirs = append(subdirs, nil)
	}

	// run the builders
	// TODO: support concurrent builds
	for i, b := range builders {
		if len(subdirs[i]) == 0 || slices.Contains(subdirs[i], ".") {
			if opts.Verbose {
				fmt.Printf("Building %s with %s\n", b.Root(), strategies[i])
			}
			if err := b.BuildAll(ctx, opts); err != nil {
				return fmt.Errorf("error building %s: %w", b.Root(), err)
			}
		} else {
			if opts.Verbose {
				fmt.Printf("Building %d dirs in %s with %s\n", len(subdirs[i]), b.Root(), strategies[i])
			}
			if err := b.BuildDirs(ctx, subdirs[i], opts); err != nil {
				return fmt.Errorf("error building %s: %w", b.Root(), err)
			}
		}
	}

	return nil
}
