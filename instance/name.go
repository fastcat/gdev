package instance

import (
	"fastcat.org/go/gdev/internal"
)

// AppName is what the app will call itself. When customizing, overwrite it
// before calling Main().
func AppName() string {
	return internal.AppName()
}

func SetAppName(name string) {
	internal.SetAppName(name)
}
