package instance

import (
	"fmt"
	"strings"
	"unicode"

	"fastcat.org/go/gdev/internal"
)

var appName = "gdev"

// AppName is what the app will call itself. When customizing, overwrite it
// before calling Main().
func AppName() string {
	return appName
}

func SetAppName(name string) {
	internal.CheckCanCustomize()
	if name == "" {
		panic(fmt.Errorf("app name must not be empty"))
	}
	if strings.ContainsFunc(name, unicode.IsSpace) {
		panic(fmt.Errorf("app name must not contain whitespace"))
	}
	appName = name
}
