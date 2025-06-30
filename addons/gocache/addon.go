package gocache

import (
	"fmt"
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
	},
}

func init() {
	addon.Definition.Initialize = initialize
}

type config struct {
	remotes []RemoteStorageFactory
}
type option func(*config)

func WithRemoteStorageFactory(f RemoteStorageFactory) option {
	if f == nil {
		panic("factory must not be nil")
	}
	return func(c *config) {
		c.remotes = append(c.remotes, f)
	}
}

func Configure(opts ...option) {
	addon.CheckNotInitialized()
	for _, o := range opts {
		o(&addon.Config)
	}
	addon.RegisterIfNeeded()
}

func initialize() error {
	instance.AddCommandBuilders(makeCmd)
	return nil
}

func makeCmd() *cobra.Command {
	var write bool
	cmd := &cobra.Command{
		Use:   "gocache [remote...]",
		Short: "Go build cache app",
		Long:  "Use with GOBUILDCACHE environment variable",
		RunE: func(cmd *cobra.Command, args []string) error {
			cd, err := os.UserCacheDir()
			if err != nil {
				return err
			}
			gbc := filepath.Join(cd, "go-build")
			// take the remotes in reverse order, first arg is most-local, last is
			// most-remote
			var remote ReadonlyStorageBackend
			canWrite := true
		ARGS:
			for i := len(args) - 1; i >= 0; i-- {
				url := args[i]
				for _, f := range addon.Config.remotes {
					if f.Want(url) {
						nextRemote, err := f.New(url)
						if err != nil {
							return fmt.Errorf("failed to create remote storage for %q: %w", url, err)
						}
						nextW, nextCanWrite := nextRemote.(StorageBackend)
						if remote == nil {
							remote = nextRemote
							canWrite = nextCanWrite
						} else if write && canWrite && nextCanWrite {
							remote = NewWriteThroughStorageBackend(nextW, remote.(StorageBackend))
						} else {
							remote = NewReadonlyStorageBackend(nextRemote, remote)
							canWrite = false
						}
						continue ARGS
					}
				}
				return fmt.Errorf("don't know how to handle remote %q", url)
			}
			var backend StorageBackend
			backend, err = DiskDirAtRoot(gbc)
			if err != nil {
				return err
			}
			if remote != nil {
				if write && canWrite {
					backend = NewWriteThroughStorageBackend(backend, remote.(StorageBackend))
				} else {
					backend = NewReadThroughStorageBackend(backend, remote)
				}
			}
			frontend := NewFrontend(backend)
			s := NewServer(frontend, os.Stdin, os.Stdout)
			// TODO: signal handlers
			return s.Run(cmd.Context())
		},
	}
	cmd.Flags().BoolVarP(&write, "write", "w", false, "enable remote write operations if possible")
	return cmd
}
