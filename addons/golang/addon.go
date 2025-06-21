package golang

import (
	"fmt"
	"io"
	"os"
	"sync"

	"fastcat.org/go/gdev/addons"
	"fastcat.org/go/gdev/addons/build"
	"fastcat.org/go/gdev/shx"
)

var addon = addons.Addon[config]{
	Definition: addons.Definition{
		Name: "golang",
		Description: func() string {
			return "Support for Go development"
		},
		Initialize: nil, // not needed here yet
	},
	Config: config{
		// TODO
	},
}

type config struct {
	// TODO
}
type option func(*config)

func Configure(opts ...option) {
	addon.CheckNotInitialized()
	for _, o := range opts {
		o(&addon.Config)
	}

	configureBuild()

	addon.RegisterIfNeeded()
}

var configureBuild = sync.OnceFunc(func() {
	build.Configure(
		build.WithStrategy("go-build", detectGoBuild, nil),
		build.WithStrategy("mage", detectMage, []string{"go-build"}),
	)
})

func buildResult(name string, res *shx.Result, err error) error {
	if err != nil {
		return fmt.Errorf("failed to start %s: %w", name, err)
	}
	defer res.Close() // nolint:errcheck
	if err = res.Err(); err != nil {
		if out := res.Stdout(); out != nil {
			_, _ = io.Copy(os.Stderr, out)
		}
		return fmt.Errorf("%s failed: %w", name, err)
	}
	if err := res.Close(); err != nil {
		return fmt.Errorf("error cleaning up after %s: %w", name, err)
	}
	return nil
}
