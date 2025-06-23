package service

import (
	"bytes"
	"fmt"
	"maps"
	"strconv"
	"strings"

	"github.com/spf13/pflag"

	"fastcat.org/go/gdev/config"
)

// TODO: configs shouldn't be globals
var serviceModes = serviceModesValue{}

func init() {
	config.AddFlag(&pflag.Flag{
		Name:     "service-modes",
		Usage:    "service modes to use for each service, in the form [\"name\"=mode,...]",
		Value:    serviceModes,
		DefValue: serviceModes.String(),
	})
}

type serviceModesValue map[string]Mode

// Set implements pflag.Value.
func (s serviceModesValue) Set(value string) error {
	clear(s)
	if value == "" {
		return nil
	}
	if value[0] != '[' {
		return fmt.Errorf("value must start with [, not %q", value[:1])
	}
	value = value[1:]
	for len(value) > 0 {
		if value[0] == ']' {
			value = value[1:]
			break
		} else if value[0] != '"' {
			return fmt.Errorf("keys must be quoted, not start with %q", value[:1])
		}
		kq, err := strconv.QuotedPrefix(value)
		if err != nil {
			return fmt.Errorf("keys must be quoted: %q", value)
		}
		value = value[len(kq):]
		k, err := strconv.Unquote(kq)
		if err != nil {
			return fmt.Errorf("invalid key quoting: %q", kq)
		}
		if value[0] != '=' {
			return fmt.Errorf("key must be followed by =, not %q", value[:1])
		}
		value = value[1:]
		modeEnd := strings.IndexAny(value, ",]")
		if modeEnd < 0 {
			return fmt.Errorf("missing delimiter after \"=\" in %q", value)
		}
		mode, ok := ParseMode(value[:modeEnd])
		if !ok {
			return fmt.Errorf("invalid mode value %q", value[:modeEnd])
		}
		if value[modeEnd] == ',' && modeEnd+2 >= len(value) {
			return fmt.Errorf("must have another entry after \",\" in %q", value[modeEnd+1:])
		}
		value = value[modeEnd+1:]
		s[k] = mode
	}
	if value != "" {
		return fmt.Errorf("value must end with ], not %q", value)
	}
	return nil
}

// String implements pflag.Value.
func (s serviceModesValue) String() string {
	// this is similar to how the "real" stringToStringValue works, but not quite
	// the same
	var buf bytes.Buffer
	buf.WriteString("[")
	for k, v := range s {
		if v == ModeDefault {
			// don't need to write these out
			continue
		}
		if buf.Len() > 1 {
			buf.WriteString(",")
		}
		// service names may need quoting
		buf.WriteString(strconv.Quote(k))
		buf.WriteString("=")
		// mode values never need quoting
		buf.WriteString(v.String())
	}
	buf.WriteString("]")
	return buf.String()
}

// Type implements pflag.Value.
func (s serviceModesValue) Type() string {
	return "stringToString"
}

func ConfiguredModes() map[string]Mode {
	return maps.Clone(serviceModes)
}

func ConfiguredMode(name string) Mode {
	if name == "" {
		panic("name must not be empty")
	}
	return serviceModes[name]
}

func SetMode(name string, mode Mode) {
	if name == "" {
		panic("name must not be empty")
	} else if !mode.Valid() {
		panic(fmt.Errorf("invalid mode %d", mode))
	}
	if serviceModes[name] != mode {
		if mode == ModeDefault {
			// remove the entry, default is implicit
			delete(serviceModes, name)
		} else {
			serviceModes[name] = mode
		}
		config.SetDirty()
	}
}
