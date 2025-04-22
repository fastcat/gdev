package main

import (
	"context"
	"fmt"

	"fastcat.org/go/gdev/magefiles/shx"
	"github.com/magefile/mage/mg"
)

func LintDefault(ctx context.Context) error {
	mg.CtxDeps(ctx, Lint{}.Golangci, Lint{}.Govulncheck)
	return nil
}

type Lint struct{}

func (Lint) Golangci(ctx context.Context) error {
	fmt.Println("Lint: golangci-lint")
	return shx.Run(ctx, "golangci-lint", "run")
}

func (Lint) Govulncheck(ctx context.Context) error {
	fmt.Println("Lint: govulncheck")
	return shx.Run(ctx, "go", "tool", "govulncheck", "./...")
}
