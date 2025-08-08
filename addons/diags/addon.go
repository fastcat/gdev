package diags

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"fastcat.org/go/gdev/addons"
	"fastcat.org/go/gdev/instance"
)

var addon = addons.Addon[config]{
	Definition: addons.Definition{
		Name:        "diags",
		Description: func() string { return "Support for diagnostics collection & uploading" },
		// Initialize:  initialize,
	},
	Config: config{
		// TODO
	},
}

func init() {
	addon.Definition.Initialize = initialize
}

type config struct {
	sources   []SourceProvider
	collector CollectorProvider
}

func Configure(opts ...option) {
	addon.CheckNotInitialized()
	for _, o := range opts {
		o(&addon.Config)
	}

	addon.RegisterIfNeeded()
}

func initialize() error {
	if len(addon.Config.sources) == 0 {
		return fmt.Errorf("no sources configured for diags addon")
	}
	if addon.Config.collector == nil {
		return fmt.Errorf("no collector configured for diags addon")
	}

	cmd := &cobra.Command{
		Use:   "diags",
		Args:  cobra.NoArgs,
		Short: "collect & upload diagnostics",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return CollectDefault(cmd.Context())
		},
	}
	instance.AddCommands(cmd)
	return nil
}

type option func(*config)

type SourceProvider func(context.Context) ([]Source, error)

func WithSources(sources ...Source) option {
	return WithSourceProvider(func(context.Context) ([]Source, error) {
		return sources, nil
	})
}

func WithSourceFuncs(sources ...SourceFunc) option {
	ret := make([]Source, 0, len(sources))
	for _, sf := range sources {
		ret = append(ret, sf)
	}
	return WithSourceProvider(func(context.Context) ([]Source, error) {
		return ret, nil
	})
}

func WithSourceProvider(sp SourceProvider) option {
	return func(c *config) {
		c.sources = append(c.sources, sp)
	}
}

type CollectorProvider func(context.Context) (Collector, error)

// No WithCollector because a collector is single-use, it must be constructed
// anew for each diags run.

func WithCollectorProvider(cp CollectorProvider) option {
	return func(cfg *config) {
		if cfg.collector != nil {
			panic("diags addon: collector already configured")
		}
		cfg.collector = cp
	}
}

// WithDefaultFileCollector returns an option that configures the diags addon to
// use a FileCollector writing to the system temp directory using
// [OpenTempDiagsFile].
//
// See: [os.TempDir]
func WithDefaultFileCollector() option {
	return WithCollectorProvider(func(ctx context.Context) (Collector, error) {
		return &TarFileCollector{
			Opener: OpenTempDiagsFile,
		}, nil
	})
}

func WithDefaultSources() option {
	return WithSourceFuncs(
		CollectAppInfo,
	)
}
