package addons

import (
	"fmt"
	"sync/atomic"

	"fastcat.org/go/gdev/internal"
)

type Definition struct {
	Name        string
	Description func() string
	Initialize  func() error
}

type registration struct {
	Definition
	initialized atomic.Bool
}
type Description struct {
	Name        string
	Description string
}

var enabled = map[string]*registration{}

func Register(def Definition) {
	if def.Name == "" {
		panic(fmt.Errorf("addon name required"))
	}
	internal.CheckCanCustomize()
	if _, ok := enabled[def.Name]; ok {
		panic(fmt.Errorf("addon %q already enabled", def.Name))
	}
	enabled[def.Name] = &registration{Definition: def}
}

func Enabled() []Description {
	internal.CheckLockedDown()
	ret := make([]Description, 0, len(enabled))
	for _, v := range enabled {
		ret = append(ret, Description{Name: v.Name, Description: v.Description()})
	}
	return ret
}
