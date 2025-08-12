package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/magefile/mage/mg"
	"golang.org/x/mod/modfile"

	"fastcat.org/go/gdev/magefiles/shx"
)

var lintOther = []any{Lint{}.Govulncheck}

func LintDefault(ctx context.Context) error {
	mg.CtxDeps(ctx, append([]any{Lint{}.Golangci}, lintOther...)...)
	return nil
}

type Lint mg.Namespace

var findGCI = sync.OnceValue(func() string {
	gb := os.Getenv("GOBIN")
	if gb == "" {
		gb = os.Getenv("GOPATH")
		if gb == "" {
			gb = os.Getenv("HOME") + "/go"
		}
		gb += "/bin"
	}
	gbInPath := false
	pathVals := os.Getenv("PATH")
	for _, dir := range filepath.SplitList(pathVals) {
		if dir == gb {
			gbInPath = true
			break
		}
	}
	if !gbInPath {
		// add GOBIN to PATH so that we can find golangci-lint
		pathVals += string(os.PathListSeparator) + gb
		_ = os.Setenv("PATH", pathVals)
	}

	if p, err := exec.LookPath("golangci-lint-v2"); err == nil {
		return p
	}
	return "golangci-lint"
})

func (Lint) Other(ctx context.Context) /* error */ {
	mg.CtxDeps(ctx, lintOther...)
	// return nil
}

func (Lint) Golangci(ctx context.Context) error {
	fmt.Println("Lint: golangci-lint")
	return shx.Cmd(ctx, findGCI(), append([]string{"run"}, modSpreads()...)...).
		With(
			// getting told the linter failed without seeing why is useless
			shx.WithOutput(),
		).
		Run()
}

func (Lint) Govulncheck(ctx context.Context) error {
	fmt.Println("Lint: govulncheck")
	return shx.Run(ctx, "go", append([]string{"tool", "govulncheck"}, modSpreads()...)...)
}

func Format(ctx context.Context) error {
	fmt.Println("Format: golangci-lint")
	return shx.Run(ctx, findGCI(), append([]string{"fmt"}, modSpreads()...)...)
}

func Tidy(ctx context.Context) error {
	w, err := workFile()
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
	w, err := workFile()
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
