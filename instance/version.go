package instance

import (
	"runtime/debug"
	"strings"
	"sync"
)

var version = sync.OnceValue(loadVersion)

func Version() string {
	return version()
}

func loadVersion() string {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return "0.0.0-development+unknown"
	}

	var rev string
	for _, s := range bi.Settings {
		switch s.Key {
		case "vcs.revision":
			rev = s.Value
			// vcs.modified not needed, go will include the +dirty for us
		}
	}

	v := bi.Main.Version
	if rev != "" {
		if len(rev) > 8 {
			rev = rev[:8]
		}
		// go revisions often contain the git hash already
		if !strings.Contains(v, rev) {
			v += "+" + rev
		}
	}
	return v
}
