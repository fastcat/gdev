package internal

import (
	"errors"
	"sync/atomic"
)

var customizationsLocked atomic.Bool

func LockCustomizations() {
	customizationsLocked.Store(true)
}

// See: [instance.CheckCanCustomize] for details and an
// importable version of this function.
func CheckCanCustomize() {
	if locked := customizationsLocked.Load(); locked {
		panic(errors.New("cannot add customizations after app start"))
	}
}

// See: [instance.CheckLockedDown] for details and an
// importable version of this function.
func CheckLockedDown() {
	if locked := customizationsLocked.Load(); !locked {
		panic(errors.New("cannot instantiate customizations until app start and lockdown"))
	}
}
