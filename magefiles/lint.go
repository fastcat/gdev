package main

import (
	"context"
	"fmt"
	"os/exec"
	"sync"

	"github.com/magefile/mage/mg"

	"fastcat.org/go/gdev/magefiles/shx"
)

func LintDefault(ctx context.Context) error {
	mg.CtxDeps(ctx, Lint{}.Golangci, Lint{}.Govulncheck)
	return nil
}

type Lint struct{}

var findGCI = sync.OnceValue(func() string {
	if p, err := exec.LookPath("golangci-lint-v2"); err == nil {
		return p
	}
	return "golangci-lint"
})

func (Lint) Golangci(ctx context.Context) error {
	fmt.Println("Lint: golangci-lint")
	return shx.Run(ctx, findGCI(), "run")
}

func (Lint) Govulncheck(ctx context.Context) error {
	fmt.Println("Lint: govulncheck")
	return shx.Run(ctx, "go", "tool", "govulncheck", "./...")
}

func Format(ctx context.Context) error {
	fmt.Println("Format: golangci-lint")
	return shx.Run(ctx, findGCI(), "fmt")
}
