package internal

import (
	"errors"
	"sync/atomic"
)

var customizationsLocked atomic.Bool

func LockCustomizations() {
	customizationsLocked.Store(true)
}

func CheckCanCustomize() {
	if locked := customizationsLocked.Load(); locked {
		panic(errors.New("cannot add customizations after app start"))
	}
}

func CheckLockedDown() {
	if locked := customizationsLocked.Load(); !locked {
		panic(errors.New("cannot instantiate customizations until app start and lockdown"))
	}
}
