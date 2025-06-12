package build

import (
	"context"
	"fmt"
	"path/filepath"

	"fastcat.org/go/gdev/service"
)

func buildBeforeStart(ctx context.Context, svc service.Service) error {
	src, ok := svc.(service.ServiceWithSource)
	if !ok {
		return nil
	}

	// TODO: service mode support, only build source if we're going to use it

	// TODO: aggregate build across all services with the same repo

	root, subDir, err := src.LocalSource(ctx)
	if err != nil {
		return fmt.Errorf("can't determine local source for service %s: %w", svc.Name(), err)
	}

	_, b, err := DetectStrategy(root)
	if err != nil {
		return fmt.Errorf("can't detect build strategy for service %s in %s: %w", svc.Name(), root, err)
	} else if b == nil {
		return fmt.Errorf("no build strategy for service %s in %s", svc.Name(), root)
	}

	if err := b.BuildDirs(ctx, []string{subDir}, Options{ /*TODO*/ }); err != nil {
		return fmt.Errorf("error building service %s in %s: %w", svc.Name(), filepath.Join(root, subDir), err)
	}

	return nil
}
