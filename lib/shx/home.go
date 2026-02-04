package shx

import (
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"sync"

	"fastcat.org/go/gdev/internal"
)

var User = sync.OnceValue(func() *user.User {
	return internal.Must(user.Current())
})

var UserName = sync.OnceValue(func() string {
	return User().Username
})

var HomeDir = sync.OnceValue(func() string {
	return internal.Must(os.UserHomeDir())
})

func PrettyPath(path string) string {
	if strings.HasPrefix(path, HomeDir()+string(filepath.Separator)) {
		path = "~" + strings.TrimPrefix(path, HomeDir())
	}
	return path
}
