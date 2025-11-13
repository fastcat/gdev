package mgx

import (
	"os"
	"os/exec"
	"path/filepath"
	"sync"
)

var FindGCI = sync.OnceValue(func() string {
	gb := os.Getenv("GOBIN")
	if gb == "" {
		gb = os.Getenv("GOPATH")
		if gb == "" {
			gb = os.Getenv("HOME") + "/go"
		}
		gb += "/bin"
	}
	gbInPath := false
	pathVals := os.Getenv("PATH")
	for _, dir := range filepath.SplitList(pathVals) {
		if dir == gb {
			gbInPath = true
			break
		}
	}
	if !gbInPath {
		// add GOBIN to PATH so that we can find golangci-lint
		pathVals += string(os.PathListSeparator) + gb
		_ = os.Setenv("PATH", pathVals)
	}

	if p, err := exec.LookPath("golangci-lint-v2"); err == nil {
		return p
	}
	return "golangci-lint"
})
