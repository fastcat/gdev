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

// CheckNotInitialized checks that the addon has not been initialized yet and
// therefore it is safe to apply customizations / configuration changes to it.
//
// If not, it panics.
//
// Includes a call to [instance.CheckCanCustomize].
func (a *addonState) CheckNotInitialized() {
	internal.CheckCanCustomize()
	if a.initialized.Load() {
		panic(errors.New("addon already initialized"))
	}
}

// CheckInitialized checks that the addon has been initialized and therefore it
// is safe to assume that its configuration is final and that it has registered
// anything it needs with the rest of the system.
//
// If not, it panics.
//
// Includes a call to [instance.CheckLockedDown].
func (a *addonState) CheckInitialized() {
	internal.CheckLockedDown()
	if !a.initialized.Load() {
		panic(errors.New("addon not initialized"))
	}
}
