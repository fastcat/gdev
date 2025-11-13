package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/magefile/mage/mg"
	"golang.org/x/mod/modfile"

	"fastcat.org/go/gdev/magefiles/mgx"
	"fastcat.org/go/gdev/magefiles/shx"
)

var lintOther = []any{Lint{}.Govulncheck}

func LintDefault(ctx context.Context) error {
	mg.CtxDeps(ctx, append([]any{Lint{}.Golangci}, lintOther...)...)
	return nil
}

type Lint mg.Namespace

func (Lint) Other(ctx context.Context) /* error */ {
	mg.CtxDeps(ctx, lintOther...)
	// return nil
}

func (Lint) Golangci(ctx context.Context) error {
	fmt.Println("Lint: golangci-lint")
	// golangci-lint doesn't support the `work` pattern
	return shx.Cmd(ctx, mgx.FindGCI(), append([]string{"run"}, mgx.ModSpreads()...)...).
		With(
			// getting told the linter failed without seeing why is useless
			shx.WithOutput(),
		).
		Run()
}

func (Lint) Govulncheck(ctx context.Context) error {
	fmt.Println("Lint: govulncheck")
	return shx.Run(ctx, "go", "tool", "govulncheck", "work")
}

func Format(ctx context.Context) error {
	fmt.Println("Format: golangci-lint")
	// golangci-lint doesn't support the `work` pattern
	return shx.Run(ctx, mgx.FindGCI(), append([]string{"fmt"}, mgx.ModSpreads()...)...)
}

func Tidy(ctx context.Context) error {
	w, err := mgx.WorkFile()
	if err != nil {
		return err
	}
	for _, m := range w.Use {
		fmt.Printf("Tidy: %s\n", m.Path)
		if err := shx.Cmd(ctx, "go", "mod", "tidy", "-v").
			With(
				shx.WithCwd(m.Path),
			).Run(); err != nil {
			return fmt.Errorf("error tidying %s: %w", m.Path, err)
		}
	}
	fmt.Println("Tidy: go work sync")
	if err := shx.Cmd(ctx, "go", "work", "sync").Run(); err != nil {
		return fmt.Errorf("error syncing go work: %w", err)
	}
	return nil
}

func SyncSelf(ctx context.Context) error {
	w, err := mgx.WorkFile()
	if err != nil {
		return err
	}
	for _, m := range w.Use {
		fmt.Printf("Sync self: %s\n", m.Path)
		mc, err := os.ReadFile(filepath.Join(m.Path, "go.mod"))
		if err != nil {
			return err
		}
		mf, err := modfile.Parse("go.mod", mc, nil)
		if err != nil {
			return err
		}
		args := []string{"get"}
		for _, r := range mf.Require {
			if r.Mod.Path == "fastcat.org/go/gdev" ||
				strings.HasPrefix(r.Mod.Path, "fastcat.org/go/gdev/") {
				args = append(args, r.Mod.Path+"@latest")
			}
		}
		if len(args) < 2 {
			continue
		}
		if err := shx.Cmd(ctx, "go", args...).
			With(shx.WithCwd(m.Path), shx.WithOutput()).
			Run(); err != nil {
			return err
		}
	}
	mg.CtxDeps(ctx, Tidy)
	return nil
}
