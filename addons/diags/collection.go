package diags

import (
	"context"
	"errors"
	"fmt"

	"golang.org/x/sync/errgroup"
)

type collection struct {
	sources []Source
	dest    Collector
}

func (c *collection) run(ctx context.Context) error {
	ctx, stop := context.WithCancel(ctx)
	defer stop()

	if err := c.dest.Begin(ctx); err != nil {
		return fmt.Errorf("error beginning collector: %w", err)
	}
	fmt.Printf("Collecting diagnostics to %s\n", c.dest.Destination())

	srcCtx, stopSrc := context.WithCancel(ctx)
	defer stopSrc()
	eg, srcCtx := errgroup.WithContext(srcCtx)
	for _, s := range c.sources {
		eg.Go(func() error {
			return s.Collect(srcCtx, c.dest)
		})
	}

	err := eg.Wait()
	stopSrc()
	if err2 := c.dest.Finalize(ctx, err); err2 != nil {
		err = errors.Join(err, err2)
	}
	return err
}

func CollectDefault(ctx context.Context) error {
	addon.CheckInitialized()
	// instantiate all the sources & the collector
	sources := make([]Source, 0, len(addon.Config.sources))
	for _, sp := range addon.Config.sources {
		sps, err := sp(ctx)
		if err != nil {
			// TODO: try to name the source provider that failed
			return fmt.Errorf("error initializing sources: %w", err)
		}
		sources = append(sources, sps...)
	}
	// we know at least one provider was configured, since the addon init will
	// panic if not.
	if len(sources) == 0 {
		return fmt.Errorf("no sources enabled")
	}
	collector, err := addon.Config.collector(ctx)
	if err != nil {
		// TODO: try to name the collector provider that failed
		return fmt.Errorf("error initializing collector: %w", err)
	}

	c := &collection{
		sources: sources,
		dest:    collector,
	}
	if len(c.sources) == 0 {
		return errors.New("no sources configured for diags addon")
	}
	return c.run(ctx)
}
