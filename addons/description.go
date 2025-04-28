package addons

import (
	"fmt"
	"maps"
	"slices"

	"fastcat.org/go/gdev/internal"
)

type Description struct {
	Name        string
	Description func() string
}

var enabled = map[string]Description{}

func AddEnabled(desc Description) {
	if desc.Name == "" {
		panic(fmt.Errorf("addon name required"))
	}
	internal.CheckCanCustomize()
	if _, ok := enabled[desc.Name]; ok {
		panic(fmt.Errorf("addon %q already enabled", desc.Name))
	}
	enabled[desc.Name] = desc
}

func Enabled() []Description {
	internal.CheckLockedDown()
	return slices.Collect(maps.Values(enabled))
}
