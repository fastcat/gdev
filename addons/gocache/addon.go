package gocache

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"fastcat.org/go/gdev/addons"
	"fastcat.org/go/gdev/instance"
)

var addon = addons.Addon[config]{
	Definition: addons.Definition{
		Name: "gocache",
		Description: func() string {
			return "Go build cache core functionality"
		},
		Initialize: initialize,
	},
}

type config struct {
	// placeholder
}

func Configure() {
	addon.CheckNotInitialized()
	addon.RegisterIfNeeded()
}

func initialize() error {
	instance.AddCommandBuilders(makeCmd)
	return nil
}

func makeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "gocache",
		Short: "Go build cache app",
		Long:  "Use with GOBUILDCACHE environment variable",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			// minimal test case: write/read-through cache
			cd, err := os.UserCacheDir()
			if err != nil {
				return err
			}
			gbc := filepath.Join(cd, "go-build")
			// second local dir
			ld := filepath.Join(cd, instance.AppName()+"-gocache")
			if err := os.Mkdir(ld, 0o750); err != nil && !errors.Is(err, os.ErrExist) {
				return err
			}
			local, err := DiskDirAtRoot(ld)
			if err != nil {
				return err
			}
			remote, err := DiskDirAtRoot(gbc)
			if err != nil {
				return err
			}
			backend := NewLayeredStorageBackend(local, remote, false)
			frontend := NewFrontend(backend)
			s := NewServer(frontend, os.Stdin, os.Stdout)
			// TODO: signal handlers
			return s.Run(cmd.Context())
		},
	}
}
