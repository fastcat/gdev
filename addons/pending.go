package addons

import (
	"fmt"

	"fastcat.org/go/gdev/internal"
)

// ordered list of addon names pending initialization
var pending []string

func Initialize() {
	// we guarantee initializers are allowed to do customizations, fail early if not
	internal.CheckCanCustomize()
	for _, name := range pending {
		addon := enabled[name]
		if addon.initialized.CompareAndSwap(false, true) {
			if err := addon.Initialize(); err != nil {
				panic(fmt.Errorf("failed to initialize addon %s: %w", addon.Name, err))
			} else if !addon.initialized.Load() {
				panic(fmt.Errorf("addon %s did not mark itself initialized", addon.Name))
			}
		}
	}
	// clear the list in case we get called again
	pending = nil
}
