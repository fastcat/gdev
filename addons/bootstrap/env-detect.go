package bootstrap

import (
	"os"
)

var isInContainerKey = NewKey[bool]("is-in-container")

func IsInContainer(ctx *Context) bool {
	if v, ok := Get(ctx, isInContainerKey); ok {
		return v
	}
	// TODO: better detection than this
	_, err := os.Stat("/.dockerenv")
	v := err == nil
	Save(ctx, isInContainerKey, v)
	return v
}

func SkipInContainer() StepOpt {
	return SkipFunc(func(ctx *Context) (bool, error) {
		return IsInContainer(ctx), nil
	})
}

var hasGUIKey = NewKey[bool]("has-gui")

func HasGUI(ctx *Context) bool {
	if v, ok := Get(ctx, hasGUIKey); ok {
		return v
	}
	v := os.Getenv("DISPLAY") != "" || os.Getenv("WAYLAND_DISPLAY") != ""
	Save(ctx, hasGUIKey, v)
	return v
}

func SkipIfNoGUI() StepOpt {
	return SkipFunc(func(ctx *Context) (bool, error) {
		return !HasGUI(ctx), nil
	})
}
