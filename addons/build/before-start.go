package build

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"fastcat.org/go/gdev/service"
	"fastcat.org/go/gdev/shx"
)

type buildBeforeStart struct {
	repoDirs map[string][]string
}

func (b *buildBeforeStart) Name() string {
	return "build-before-start"
}

func (b *buildBeforeStart) BeforeServices(ctx context.Context, infra, svcs []service.Service) error {
	// make sure initial state is clean
	b.repoDirs = make(map[string][]string)
	return nil
}

func (b *buildBeforeStart) Service(ctx context.Context, svc service.Service) error {
	src, ok := svc.(service.ServiceWithSource)
	if !ok {
		return nil // Not a source service, nothing to do
	}
	// TODO: use service mode instead of dir existence to decide whether to build it
	root, subDir, err := src.LocalSource(ctx)
	if err != nil {
		return fmt.Errorf("can't determine local source for service %s: %w", svc.Name(), err)
	}
	if _, err := os.Stat(filepath.Join(root, subDir)); err != nil {
		if os.IsNotExist(err) {
			// skip the build
			return nil
		}
		return fmt.Errorf("error checking local source for service %s in %s: %w",
			svc.Name(), filepath.Join(root, subDir), err,
		)
	}
	// normalize root dirs to ensure map keys are consistent
	root, err = filepath.Abs(root)
	if err != nil {
		return fmt.Errorf("can't get absolute path for service %s in %s: %w", svc.Name(), root, err)
	}

	// clean subdir path too so we can clear dupes later
	b.repoDirs[root] = append(b.repoDirs[root], filepath.Clean(subDir))

	return nil
}

func (b *buildBeforeStart) AfterServices(ctx context.Context, infra, svcs []service.Service) error {
	// TODO: build each repo concurrently
	for root, subDirs := range b.repoDirs {
		// TODO: deduplicate subDirs

		prettyRoot := shx.PrettyPath(root)

		sn, b, err := DetectStrategy(root)
		if err != nil {
			return fmt.Errorf("can't detect build strategy repo %s: %w", prettyRoot, err)
		} else if b == nil {
			return fmt.Errorf("no build strategy for repo %s", prettyRoot)
		}
		opts := Options{ /* TODO */ }
		// if any service needs the repo root, use BuildAll
		if slices.Contains(subDirs, "") {
			fmt.Printf("Building %s using %s\n", prettyRoot, sn)
			err = b.BuildAll(ctx, opts)
		} else {
			fmt.Printf("Building %s using %s with subdirs %v\n", prettyRoot, sn, subDirs)
			err = b.BuildDirs(ctx, subDirs, opts)
		}
		if err != nil {
			return fmt.Errorf("error building repo %s with strategy %s: %w", prettyRoot, sn, err)
		}
	}
	return nil
}
