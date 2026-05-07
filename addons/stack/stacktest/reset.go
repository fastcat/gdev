package stacktest

import (
	sInternal "fastcat.org/go/gdev/addons/stack/internal"
	"fastcat.org/go/gdev/internal"
)

// ResetServices unlocks and resets (clears) the registered stack services.
//
// Only use this in test harnesses, and be careful.
//
// CLI commands generated from the registered services will not be reset /
// regenerated, so the uses of this are limited.
//
// Running this while the stack is running WILL cause problems.
func ResetServices() {
	internal.CheckLockedDown()
	sInternal.Reset()
}
