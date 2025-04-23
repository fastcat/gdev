package main

import (
	"context"
	"fmt"

	"fastcat.org/go/gdev/magefiles/shx"
)

func Test(ctx context.Context) error {
	fmt.Println("Test: go test -race")
	return shx.Run(ctx, "go", "test", "-race", "./...")
}
