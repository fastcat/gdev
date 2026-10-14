package bootstrap

import (
	"fmt"
	"reflect"
	"strconv"

	"github.com/spf13/pflag"

	"fastcat.org/go/gdev/addons/bootstrap/internal"
)

var skipLoginsKey = NewKey[bool]("bootstrap.skipLogins")

// SkipLogins returns whether login steps should be skipped. This is set by a
// command line argument. Custom bootstrap steps/plans should obey this.
func SkipLogins(ctx *Context) bool {
	v, ok := ctx.Get(skipLoginsKey)
	return ok && v
}

func SkipIfNoLogins() StepOpt {
	return SkipFunc(func(ctx *Context) (bool, error) {
		return SkipLogins(ctx), nil
	})
}

var headlessKey = NewKey[bool]("bootstrap.headless")

// Headless returns whether steps should run in headless mode. This is set by a
// command line argument. Custom bootstrap steps/plans should obey this.
func Headless(ctx *Context) bool {
	v, ok := ctx.Get(headlessKey)
	return ok && v
}

// WithKeyFlag defines a custom CLI flag for all bootstrap commands that allows
// setting an initial default value for the given context info key.
func WithKeyFlag[
	T interface{ bool | int | uint | string },
](
	key InfoKey[T],
	flagName, help string,
) Option {
	var handler func(string) error
	var installer func(f *pflag.FlagSet, flagName, help string)
	switch key := any(key).(type) {
	case InfoKey[bool]:
		handler = func(s string) error {
			if s == "" {
				internal.SetDefault(key, true)
			} else if v, err := strconv.ParseBool(s); err != nil {
				return err
			} else {
				internal.SetDefault(key, v)
			}
			return nil
		}
		installer = func(f *pflag.FlagSet, flagName, help string) {
			f.BoolFunc(flagName, help, handler)
		}
	case InfoKey[int]:
		handler = func(s string) error {
			if v, err := strconv.ParseInt(s, 0, int(reflect.TypeFor[int]().Size()*8)); err != nil {
				return err
			} else {
				internal.SetDefault(key, int(v))
				return nil
			}
		}
	case InfoKey[uint]:
		handler = func(s string) error {
			if v, err := strconv.ParseUint(s, 0, int(reflect.TypeFor[uint]().Size()*8)); err != nil {
				return err
			} else {
				internal.SetDefault(key, uint(v))
				return nil
			}
		}
	case InfoKey[string]:
		handler = func(s string) error { //nolint:unparam
			internal.SetDefault(key, s)
			return nil
		}
	}
	if installer == nil {
		installer = func(f *pflag.FlagSet, flagName, help string) {
			f.Func(flagName, help, handler)
		}
	}
	return func(c *config) {
		if c.keyFlags == nil {
			c.keyFlags = make(map[string]func(*pflag.FlagSet))
		} else if _, ok := c.keyFlags[flagName]; ok {
			panic(fmt.Sprintf("key flag %q already defined", flagName))
		}
		c.keyFlags[flagName] = func(f *pflag.FlagSet) { installer(f, flagName, help) }
	}
}
