package addons

import (
	"fmt"

	"fastcat.org/go/gdev/internal"
)

func Initialize() {
	// we guarantee initializers are allowed to do customizations, fail early if not
	internal.CheckCanCustomize()
	for _, d := range enabled {
		if d.initialized.CompareAndSwap(false, true) {
			if err := d.Initialize(); err != nil {
				panic(fmt.Errorf("failed to initialize addon %s: %w", d.Name, err))
			}
		}
	}
}
