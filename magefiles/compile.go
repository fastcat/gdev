package main

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/magefile/mage/mg"

	"fastcat.org/go/gdev/magefiles/mgx"
	"fastcat.org/go/gdev/magefiles/shx"
)

func Compile(ctx context.Context) error {
	fmt.Println("Compile: go build")
	// have to tell go each module, else it will only build the root module
	args := []string{"build", "-v"}
	args = append(args, mgx.ModSpreads("./magefiles")...)
	return shx.Run(ctx, "go", args...)
}

type Build mg.Namespace

func (Build) Debug(ctx context.Context) /* error */ {
	mg.CtxDeps(ctx, mg.F(Build{}.debug, "./examples/gdev", "./gdev.debug"))
	// return nil
}

func (Build) Release(ctx context.Context) /* error */ {
	mg.CtxDeps(ctx, mg.F(Build{}.release, "./examples/gdev", "./gdev"))
	// return nil
}

func (Build) debug(ctx context.Context, pkg, name string) error {
	fmt.Printf("Build %s debug binary\n", filepath.Base(pkg))
	return shx.Cmd(
		ctx,
		"go", "build", "-gcflags=all=-N -l", "-v", "-o", name, pkg,
	).Run()
}

func (Build) release(ctx context.Context, pkg, name string) error {
	fmt.Printf("Build %s release binary\n", filepath.Base(pkg))
	return shx.Cmd(
		ctx,
		"go", "build", "-ldflags=-s -w", "-v", "-o", name, pkg,
	).With(
		shx.WithEnv(map[string]string{
			"CGO_ENABLED": "0",
		}),
	).Run()
}

// Build release binaries for each example app
func (Build) Examples(ctx context.Context) /* error */ {
	mg.CtxDeps(ctx,
		mg.F(Build{}.release, "./examples/gdev", "./gdev"),
		mg.F(Build{}.release, "./examples/custom-commands", "./example-custom-commands"),
		mg.F(Build{}.release, "./examples/stack", "./example-stack"),
	)
	// return nil
}
