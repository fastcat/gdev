package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/magefile/mage/mg"

	"fastcat.org/go/gdev/magefiles/shx"
)

var lintOther = []any{Lint{}.Govulncheck}

func LintDefault(ctx context.Context) error {
	mg.CtxDeps(ctx, append([]any{Lint{}.Golangci}, lintOther...)...)
	return nil
}

type Lint mg.Namespace

var findGCI = sync.OnceValue(func() string {
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

func (Lint) Other(ctx context.Context) error {
	mg.CtxDeps(ctx, lintOther...)
	return nil
}

func (Lint) Golangci(ctx context.Context) error {
	fmt.Println("Lint: golangci-lint")
	return shx.Cmd(ctx, findGCI(), "run").
		With(
			// getting told the linter failed without seeing why is useless
			shx.WithOutput(),
		).
		Run()
}

func (Lint) Govulncheck(ctx context.Context) error {
	fmt.Println("Lint: govulncheck")
	return shx.Run(ctx, "go", "tool", "govulncheck", "./...")
}

func Format(ctx context.Context) error {
	fmt.Println("Format: golangci-lint")
	return shx.Run(ctx, findGCI(), "fmt")
}
