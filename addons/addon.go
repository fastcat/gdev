package addons

import (
	"errors"
	"sync/atomic"

	"fastcat.org/go/gdev/internal"
)

type addonState struct {
	registered  atomic.Bool
	initialized atomic.Bool
}

type Addon[T any] struct {
	addonState
	Config     T
	Definition Definition
}

func (a *Addon[T]) RegisterIfNeeded() {
	if a.registered.CompareAndSwap(false, true) {
		Register(a)
	}
}

func (a *addonState) CheckNotInitialized() {
	internal.CheckCanCustomize()
	if a.initialized.Load() {
		panic(errors.New("addon already initialized"))
	}
}

func (a *addonState) CheckInitialized() {
	internal.CheckLockedDown()
	if !a.initialized.Load() {
		panic(errors.New("addon not initialized"))
	}
}
