package service

import (
	"fmt"

	"fastcat.org/go/gdev/lib/config"
)

func init() {
	config.AddKey(serviceModesKey{})
	// Usage:    "service modes to use for each service, in the form [\"name\"=mode,...]",
}

type serviceModesKey struct{}

// Name implements config.ConfigKey.
func (s serviceModesKey) Name() string {
	return "service-modes"
}

// IsDefault implements config.ConfigKey.
func (s serviceModesKey) IsDefault(value map[string]Mode) bool {
	return len(value) == 0
}

// New implements config.ConfigKey.
func (s serviceModesKey) New() map[string]Mode {
	return map[string]Mode{}
}

// NewFrom implements config.ConfigKey.
func (s serviceModesKey) NewFrom(value any) (map[string]Mode, error) {
	m, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("expected map[string]any for service-modes, got %T", value)
	}
	sm := make(map[string]Mode, len(m))
	for k, v := range m {
		if mode, ok := v.(string); !ok {
			return nil, fmt.Errorf("expected string for service-modes[%q], got %T", k, v)
		} else if sm[k], ok = ParseMode(mode); !ok {
			return nil, fmt.Errorf("invalid mode %q for service-modes[%q]", mode, k)
		}
	}
	return sm, nil
}

func ConfiguredMode(name string) Mode {
	if name == "" {
		panic("name must not be empty")
	}
	sm := config.Get(serviceModesKey{})
	return sm[name]
}

func SetMode(name string, mode Mode) {
	if name == "" {
		panic("name must not be empty")
	} else if !mode.Valid() {
		panic(fmt.Errorf("invalid mode %d", mode))
	}
	sm := config.Get(serviceModesKey{})
	if sm[name] != mode {
		if mode == ModeDefault {
			// remove the entry, default is implicit
			delete(sm, name)
		} else {
			sm[name] = mode
		}
		config.SetDirty()
	}
}
