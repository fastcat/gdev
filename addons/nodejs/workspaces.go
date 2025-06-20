package nodejs

import (
	"fmt"
	"path/filepath"
	"strings"
)

func expandWorkspaces(root string, globs []string) ([]string, error) {
	if len(globs) == 0 {
		return nil, nil
	}
	workspaces := make([]string, 0, len(globs))
	for _, w := range globs {
		var matches []string
		var err error
		if !strings.Contains(w, "*") {
			// simple path
			matches = []string{filepath.Join(root, w)}
		} else {
			// glob pattern, expand it
			globPath := filepath.Join(root, w)
			if matches, err = filepath.Glob(globPath); err != nil {
				return nil, fmt.Errorf("failed to expand glob pattern %s: %w", globPath, err)
			} else if len(matches) == 0 {
				return nil, fmt.Errorf("no matches found for glob pattern %s", globPath)
			}
		}
		for _, match := range matches {
			subdir, err := filepath.Rel(root, match)
			if err != nil {
				return nil, fmt.Errorf("failed to get relative workspace path for %s: %w", match, err)
			}
			workspaces = append(workspaces, subdir)
		}
	}
	return workspaces, nil
}
