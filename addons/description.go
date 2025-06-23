package addons

import (
	"fmt"

	"fastcat.org/go/gdev/internal"
)

type Definition struct {
	Name        string
	Description func() string
	Initialize  func() error
}

type registration struct {
	Definition
	state *addonState
}
type Description struct {
	Name        string
	Description string
}

var enabled = map[string]*registration{}

func Register[T any](a *Addon[T]) {
	if a.Definition.Name == "" {
		panic(fmt.Errorf("addon name required"))
	}
	if a.Definition.Description == nil {
		panic(fmt.Errorf("addon %q requires a description", a.Definition.Name))
	}
	internal.CheckCanCustomize()
	if _, ok := enabled[a.Definition.Name]; ok {
		panic(fmt.Errorf("addon %q already enabled", a.Definition.Name))
	}
	enabled[a.Definition.Name] = &registration{Definition: a.Definition, state: &a.addonState}
	pending = append(pending, a.Definition.Name)
}

func Enabled() []Description {
	internal.CheckLockedDown()
	ret := make([]Description, 0, len(enabled))
	for _, v := range enabled {
		ret = append(ret, Description{Name: v.Name, Description: v.Description()})
	}
	return ret
}
