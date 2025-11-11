package main

import (
	"context"
	"fmt"
	"os"

	"fastcat.org/go/gdev/magefiles/shx"
)

func Test(ctx context.Context) error {
	fmt.Println("Test: go test -race")
	args := []string{"test", "-race", "-timeout", "30s"}
	if os.Getenv("VERBOSE") != "" || os.Getenv("CI") != "" {
		args = append(args, "-v")
	}
	args = append(args, "./...")
	return shx.Run(ctx, "go", args...)
}
