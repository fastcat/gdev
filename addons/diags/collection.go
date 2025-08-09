package diags

import (
	"context"
	"errors"
	"fmt"

	"golang.org/x/sync/errgroup"
)

type Collection struct {
	Sources []Source
	Dest    Collector
}

func (c *Collection) Run(ctx context.Context) error {
	ctx, stop := context.WithCancel(ctx)
	defer stop()

	if err := c.Dest.Begin(ctx); err != nil {
		return fmt.Errorf("error beginning collector: %w", err)
	}
	fmt.Printf("Collecting diagnostics to %s\n", c.Dest.Destination())

	srcCtx, stopSrc := context.WithCancel(ctx)
	defer stopSrc()
	eg, srcCtx := errgroup.WithContext(srcCtx)
	for _, s := range c.Sources {
		eg.Go(func() error {
			return s.Collect(srcCtx, c.Dest)
		})
	}

	err := eg.Wait()
	stopSrc()
	if err2 := c.Dest.Finalize(ctx, err); err2 != nil {
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

	c := &Collection{
		Sources: sources,
		Dest:    collector,
	}
	if len(c.Sources) == 0 {
		return errors.New("no sources configured for diags addon")
	}
	return c.Run(ctx)
}
