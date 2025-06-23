package config

import (
	"fmt"
	"sync/atomic"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"fastcat.org/go/gdev/instance"
	"fastcat.org/go/gdev/internal"
)

var (
	vi           *viper.Viper
	pendingFlags = map[string]*pflag.Flag{}
	dirty        atomic.Int32
)

func Initialize() error {
	internal.CheckLockedDown()
	if vi != nil {
		panic("config already initialized")
	}
	vi = viper.New()
	vi.SetEnvPrefix(instance.AppName())
	vi.AutomaticEnv()
	// TODO: let app instance customize this
	vi.SetConfigType("yaml")

	for k, f := range pendingFlags {
		if err := vi.BindPFlag(k, f); err != nil {
			return fmt.Errorf("failed to bind flag %q: %w", k, err)
		}
	}
	pendingFlags = nil

	if err := vi.ReadInConfig(); err != nil {
		return err
	}
	return nil
}

func AddFlag(
	f *pflag.Flag,
) {
	if _, ok := pendingFlags[f.Name]; ok {
		panic(fmt.Errorf("flag %q already registered", f.Name))
	}
	pendingFlags[f.Name] = f
}

func SetDirty() {
	dirty.Add(1)
}

func IsDirty() bool {
	return dirty.Load() > 0
}
