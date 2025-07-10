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
	factories      []RemoteStorageFactory
	defaultRemotes []string
}
type option func(*config)

func WithRemoteStorageFactory(f RemoteStorageFactory) option {
	if f == nil {
		panic("factory must not be nil")
	}
	return func(c *config) {
		c.factories = append(c.factories, f)
	}
}

func WithDefaultRemotes(remotes ...string) option {
	if len(remotes) == 0 {
		panic("remotes must not be empty")
	}
	return func(c *config) {
		for _, r := range remotes {
			ok := false
			for _, f := range c.factories {
				if f.Want(r) {
					ok = true
					break
				}
			}
			if !ok {
				panic(fmt.Sprintf("remote %q is not supported by any registered factory", r))
			}
		}
		c.defaultRemotes = append(c.defaultRemotes, remotes...)
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
	writeThrough := true
	withDefaults := true

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
			if withDefaults {
				expanded := make([]string, 0, len(addon.Config.defaultRemotes)+len(args))
				expanded = append(expanded, addon.Config.defaultRemotes...)
				expanded = append(expanded, args...)
				args = expanded
			}
		ARGS:
			for i := len(args) - 1; i >= 0; i-- {
				url := args[i]
				for _, f := range addon.Config.factories {
					if f.Want(url) {
						nextRemote, err := f.New(url)
						if err != nil {
							return fmt.Errorf("failed to create remote storage for %q: %w", url, err)
						}
						nextW, nextCanWrite := nextRemote.(StorageBackend)
						if remote == nil {
							remote = nextRemote
							canWrite = nextCanWrite
							// if canWrite {
							// 	fmt.Fprintln(os.Stderr, "remote write", url)
							// } else {
							// 	fmt.Fprintln(os.Stderr, "remote read", url)
							// }
						} else if writeThrough && canWrite && nextCanWrite {
							// fmt.Fprintln(os.Stderr, "remote write-through", url)
							remote = NewWriteThroughStorageBackend(nextW, remote.(StorageBackend))
						} else {
							// fmt.Fprintln(os.Stderr, "remote read-only", url)
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
				if writeThrough && canWrite {
					// fmt.Fprintln(os.Stderr, "final write-through", gbc)
					backend = NewWriteThroughStorageBackend(backend, remote.(StorageBackend))
				} else {
					// fmt.Fprintln(os.Stderr, "final read-through", remote)
					backend = NewReadThroughStorageBackend(backend, remote)
				}
			}
			frontend := NewFrontend(backend)
			s := NewServer(frontend, os.Stdin, os.Stdout)
			// time.Sleep(5 * time.Second)
			// TODO: signal handlers
			return s.Run(cmd.Context())
		},
	}
	f := cmd.Flags()
	f.BoolVarP(&writeThrough, "write", "w", writeThrough,
		"enable remote write operations if possible")
	f.BoolVar(&withDefaults, "with-defaults", withDefaults,
		fmt.Sprintf("include default remotes (%d) in the list of remotes to use",
			len(addon.Config.defaultRemotes)),
	)
	return cmd
}
