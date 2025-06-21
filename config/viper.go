package config

import (
	"github.com/spf13/viper"

	"fastcat.org/go/gdev/instance"
	"fastcat.org/go/gdev/internal"
)

var vi *viper.Viper

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
	if err := vi.ReadInConfig(); err != nil {
		return err
	}
	return nil
}
