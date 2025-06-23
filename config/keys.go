package config

import (
	"fmt"
	"sync/atomic"
)

var (
	keys  = map[string]anyConfigKey{}
	data  map[string]any // set non-nil once in Initialize
	dirty atomic.Int32
)

type anyConfigKey struct {
	key       interface{ Name() string }
	isDefault func(value any) bool
	new       func() any
	newFrom   func(value any) (any, error)
}

type ConfigKey[T any] interface {
	Name() string
	New() T
	NewFrom(value any) (T, error)
	IsDefault(value T) bool
	// TODO: support direct unmarshal from YAML
}

func AddKey[T any](key ConfigKey[T]) {
	name := key.Name()
	if _, ok := keys[name]; ok {
		panic(fmt.Errorf("config key %q already registered", name))
	}
	keys[name] = anyConfigKey{
		key,
		func(value any) bool { return key.IsDefault(value.(T)) },
		func() any { return key.New() },
		func(value any) (any, error) { return key.NewFrom(value) },
	}
}

func SetDirty() {
	dirty.Add(1)
}

func IsDirty() bool {
	return dirty.Load() > 0
}

func Get[T any](key ConfigKey[T]) T {
	if keys[key.Name()].key != key {
		panic(fmt.Errorf("incorrect config key for %q", key.Name()))
	}
	return data[key.Name()].(T)
}
