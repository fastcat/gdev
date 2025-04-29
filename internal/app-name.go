package internal

import (
	"fmt"
	"strings"
	"sync/atomic"
	"unicode"
)

var appName = "gdev"

// we need to be able to access the app name early in a lot of places, so it has
// its own lockdown tracker in addition to the main one
var appNameLocked atomic.Bool

// AppName is what the app will call itself. When customizing, overwrite it
// before calling Main().
func AppName() string {
	// once observed it cannot be changed
	appNameLocked.Store(true)
	return appName
}

func SetAppName(name string) {
	CheckCanCustomize()
	if appNameLocked.Load() {
		panic(fmt.Errorf("app name is locked"))
	}
	if name == "" {
		panic(fmt.Errorf("app name must not be empty"))
	}
	if strings.ContainsFunc(name, unicode.IsSpace) {
		panic(fmt.Errorf("app name must not contain whitespace"))
	}
	appName = name
}
