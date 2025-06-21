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
		addonReg := enabled[name]
		if addonReg.state.initialized.Load() {
			// already initialized
			continue
		}
		if addonReg.Initialize != nil {
			if err := addonReg.Initialize(); err != nil {
				panic(fmt.Errorf("failed to initialize addon %s: %w", addonReg.Name, err))
			}
		}
		addonReg.state.initialized.Store(true)
	}
	// clear the list in case we get called again
	pending = nil
}
