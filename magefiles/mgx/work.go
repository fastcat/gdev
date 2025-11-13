package mgx

import (
	"os"
	"slices"
	"strings"
	"sync"

	"golang.org/x/mod/modfile"
)

var WorkFile = sync.OnceValues(func() (*modfile.WorkFile, error) {
	if wc, err := os.ReadFile("./go.work"); err != nil {
		return nil, err
	} else if w, err := modfile.ParseWork("go.work", wc, nil); err != nil {
		return nil, err
	} else {
		return w, nil
	}
})

// Generate `./dir/...` for each module in the work file, except the one(s)
// listed.
//
// Any path that starts with `../` is always skipped.
//
// If you don't need to exclude any, then use the `work` pattern instead if the
// tool supports it.
func ModSpreads(exclude ...string) []string {
	w, err := WorkFile()
	if err != nil {
		panic(err)
	}
	spreads := make([]string, 0, len(w.Use))
	for _, m := range w.Use {
		if strings.HasPrefix(m.Path, "../") {
			continue
		} else if slices.Contains(exclude, m.Path) {
			continue
		}
		spreads = append(spreads, m.Path+"/...")
	}
	return spreads
}
