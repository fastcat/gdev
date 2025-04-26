package main

import (
	"context"

	"github.com/magefile/mage/mg"
)

var (
	Default = All
	Aliases = map[string]any{
		"lint": LintDefault,
		"fmt":  Format,
	}
)

func All(ctx context.Context) error {
	mg.CtxDeps(ctx, LintDefault, Compile, Test)
	return nil
}
