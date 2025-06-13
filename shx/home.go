package shx

import (
	"os"
	"path/filepath"
	"strings"
	"sync"

	"fastcat.org/go/gdev/internal"
)

var HomeDir = sync.OnceValue(func() string {
	return internal.Must(os.UserHomeDir())
})

func PrettyPath(path string) string {
	if strings.HasPrefix(path, HomeDir()+string(filepath.Separator)) {
		path = "~" + strings.TrimPrefix(path, HomeDir())
	}
	return path
}
