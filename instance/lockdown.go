package instance

import (
	"os"

	"fastcat.org/go/gdev/internal"
)

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

// TestMain is an implementation you can use for your own TestMain, to get the
// app initialized before running the tests.
//
// Use this if you're struggling in tests with
// "cannot instantiate customizations until app start and lockdown"
//
// You still need to call [instance.SetAppName] before calling this.
func TestMain(m interface{ Run() int }) {
	internal.LockCustomizations()
	os.Exit(m.Run()) //nolint:forbidigo // entrypoint
}
