package mgx

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
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
	if slices.Contains(filepath.SplitList(pathVals), gb) {
		gbInPath = true
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
