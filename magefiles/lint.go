package main

import (
	"context"
	"fmt"
	"os/exec"
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
