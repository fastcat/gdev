package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/goccy/go-yaml" //cspell:ignore goccy

	"fastcat.org/go/gdev/magefiles/mgx"
)

func GenerateVanityFiles(_ context.Context, root string) error {
	w, err := mgx.WorkFile()
	if err != nil {
		return err
	}
	const rootStart = `<meta name="go-import" content="fastcat.org/go/gdev`
	const rootContent = rootStart + ` git https://github.com/fastcat/gdev.git">` + "\n"
	const template = rootStart + `/%[1]s git https://github.com/fastcat/gdev.git %[1]s">` + "\n"
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

func UpdateDependabotConfig(_ context.Context) error {
	content, err := os.ReadFile(".github/dependabot.yaml")
	if err != nil {
		return err
	}
	cm := yaml.CommentMap{}
	var parsed yaml.MapSlice
	if err := yaml.UnmarshalWithOptions(content, &parsed,
		yaml.CommentToMap(cm),
		yaml.UseOrderedMap(),
	); err != nil {
		return fmt.Errorf("error parsing dependabot config: %w", err)
	}
	var updates []any
	updatesIdx := -1
	for idx, item := range parsed {
		if item.Key == "updates" {
			updatesIdx, updates = idx, item.Value.([]any)
			break
		}
	}
	if updates == nil {
		return fmt.Errorf("no updates section found in dependabot config")
	}
	// find the root as a template
	var root yaml.MapSlice
	have := map[string]bool{}
	for _, u := range updates {
		u := u.(yaml.MapSlice)
		um := u.ToMap()
		if um["package-ecosystem"] != "gomod" { //cspell:ignore gomod
			continue
		} else if dir := um["directory"].(string); dir == "/" {
			root = u
		} else {
			have[dir] = true
		}
	}
	if root == nil {
		return fmt.Errorf("no root update found in dependabot config")
	}
	// add copies of the root for any mods we don't already have
	w, err := mgx.WorkFile()
	if err != nil {
		return err
	}
	changed := false
	for _, use := range w.Use {
		if use.Path == "." {
			continue
		}
		pp := "/" + strings.TrimPrefix(use.Path, "./")
		if have[pp] {
			continue
		}
		newUpdate := slices.Clone(root)
		for idx, kv := range newUpdate {
			if kv.Key == "directory" {
				kv.Value = pp
				newUpdate[idx] = kv
				break
			}
		}
		updates = append(updates, newUpdate)
		changed = true
	}
	if changed {
		parsed[updatesIdx].Value = updates
		if content, err := yaml.MarshalWithOptions(parsed, yaml.WithComment(cm)); err != nil {
			return err
		} else if err := os.WriteFile(".github/dependabot.yaml", content, 0o644); err != nil {
			return err
		}
	}
	return nil
}
