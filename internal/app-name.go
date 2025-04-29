package internal

import (
	"fmt"
	"strings"
	"unicode"
)

var appName = "gdev"

// AppName is what the app will call itself. When customizing, overwrite it
// before calling Main().
func AppName() string {
	CheckLockedDown()
	return appName
}

func SetAppName(name string) {
	CheckCanCustomize()
	if name == "" {
		panic(fmt.Errorf("app name must not be empty"))
	}
	if strings.ContainsFunc(name, unicode.IsSpace) {
		panic(fmt.Errorf("app name must not contain whitespace"))
	}
	appName = name
}
