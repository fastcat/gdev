package instance

import "fastcat.org/go/gdev/internal"

// re-exports of some internal functions that we want to be accessible. the main
// reason for hiding the "original" versions of these is to hide the mutator.

// CheckCanCustomize panics if customizations have been locked down at this
// point in the app startup process.
//
// Use this as a guard / assertion in functions that try to change
// customizations to ensure they are not called from the wrong place where they
// may be ineffectual or cause errors.
func CheckCanCustomize() {
	internal.CheckCanCustomize()
}

// CheckLockedDown panics if customizations have not been locked down at this
// point in the app startup process.
//
// Use this as a guard / assertion in functions that put customizations into
// effect to ensure that they are not called from places where customizations
// may be modified later and thus break or prevent those future modifications.
func CheckLockedDown() {
	internal.CheckLockedDown()
}
