package main

import (
	"context"
	"fmt"

	"fastcat.org/go/gdev/magefiles/shx"
)

func Compile(ctx context.Context) error {
	fmt.Println("Compile: go build")
	return shx.Run(ctx, "go", "build", "-v", "./...")
}
