package addons

import (
	"errors"
	"sync/atomic"

	"fastcat.org/go/gdev/internal"
)

type Addon[T any] struct {
	Config      T
	registered  atomic.Bool
	initialized atomic.Bool
}

func (a *Addon[T]) RegisterIfNeeded(def Definition) {
	if a.registered.CompareAndSwap(false, true) {
		Register(def)
	}
}

func (a *Addon[T]) CheckNotInitialized() {
	internal.CheckCanCustomize()
	if a.initialized.Load() {
		panic(errors.New("addon already initialized"))
	}
}

func (a *Addon[T]) CheckInitialized() {
	internal.CheckLockedDown()
	if !a.initialized.Load() {
		panic(errors.New("addon not initialized"))
	}
}

func (a *Addon[T]) Initialized() {
	if !a.registered.Load() {
		panic(errors.New("initializing addon without registering"))
	}
	a.initialized.Store(true)
}
