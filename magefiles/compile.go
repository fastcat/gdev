package main

import (
	"context"
	"fmt"

	"github.com/magefile/mage/mg"

	"fastcat.org/go/gdev/magefiles/shx"
)

func Compile(ctx context.Context) error {
	fmt.Println("Compile: go build")
	// have to tell go each module, else it will only build the root module
	w, err := workFile()
	if err != nil {
		return err
	}
	args := []string{"build", "-v"}
	for _, m := range w.Use {
		if m.Path == "./magefiles" {
			continue
		}
		args = append(args, m.Path+"/...")
	}
	return shx.Run(ctx, "go", args...)
}

type Build mg.Namespace

func (Build) Debug(ctx context.Context) error {
	fmt.Println("Build gdev debug binary")
	return shx.Run(ctx, "go", "build", "-gcflags=-N -l", "-v", "-o", "./gdev.debug", "./examples/gdev")
}

func (Build) Release(ctx context.Context) error {
	fmt.Println("Build gdev release binary")
	return shx.Cmd(
		ctx,
		"go", "build", "-ldflags=-s -w", "-v", "-o", "./gdev", "./examples/gdev",
	).With(
		shx.WithEnv(map[string]string{
			"CGO_ENABLED": "0",
		}),
	).Run()
}
