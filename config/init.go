package config

import (
	"fmt"

	"fastcat.org/go/gdev/internal"
)

func Initialize() error {
	internal.CheckLockedDown()
	if data != nil {
		panic(fmt.Errorf("config already initialized"))
	}
	data = make(map[string]any, len(keys))
	for k, kk := range keys {
		data[k] = kk.new()
	}
	return load()
}
