package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/mod/modfile"
)

func GenerateVanityFiles(ctx context.Context, root string) error {
	wc, err := os.ReadFile("./go.work")
	if err != nil {
		return err
	}
	w, err := modfile.ParseWork("go.work", wc, nil)
	if err != nil {
		return err
	}
	const rootContent = `<meta name="go-import" content="fastcat.org/go/gdev git https://github.com/fastcat/gdev">` + "\n"
	const template = `<meta name="go-import" content="fastcat.org/go/gdev/%[1]s git https://github.com/fastcat/gdev %[1]s">` + "\n"
	for _, use := range w.Use {
		var content string
		if use.Path == "." {
			content = rootContent
		} else {
			content = fmt.Sprintf(template, strings.TrimPrefix(use.Path, "./"))
		}
		dest := filepath.Join(root, use.Path, "index.html")
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return err
		} else if err := os.WriteFile(dest, []byte(content), 0o644); err != nil {
			return err
		}
	}

	return nil
}
