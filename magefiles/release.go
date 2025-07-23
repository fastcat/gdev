package main

import (
	"context"
	"fmt"
	"path"
	"strings"

	"golang.org/x/mod/semver"

	"fastcat.org/go/gdev/magefiles/shx"
)

func TagSubModules(ctx context.Context, newVersion string) error {
	if !semver.IsValid(newVersion) {
		return fmt.Errorf("invalid version: %q", newVersion)
	}
	w, err := workFile()
	if err != nil {
		return err
	}

	for _, m := range w.Use {
		p := path.Clean(m.Path)
		if p == "." {
			// root module is what (semrel) just tagged
			continue
		} else if p == "magefiles" {
			// magefiles is not "published"
			continue
		} else if strings.HasPrefix(p, "examples/") {
			// examples are not "published"
			continue
		}
		// copy the root tag to the submodule. force lightweight tags so that local
		// runs match CI runs and don't prompt for a message due to signing.
		if err := shx.Run(ctx,
			"git", "tag", "--no-sign", p+"/"+newVersion, newVersion,
		); err != nil {
			return err
		}
	}
	return nil
}
