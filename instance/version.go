package instance

import (
	"runtime/debug"
	"strings"
	"sync"
)

var info = sync.OnceValue(loadVersionInfo)

func Version() string {
	return info().MainVersion
}

func VersionInfo() versionInfo {
	return info()
}

type versionInfo struct {
	MainVersion string
	MainModule  string
	MainRev     string

	GDevVersion string
}

func loadVersionInfo() versionInfo {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return versionInfo{
			MainVersion: "0.0.0-development+unknown",
			MainModule:  "unknown",
			GDevVersion: "0.0.0-development+unknown",
		}
	}
	var ret versionInfo

	for _, s := range bi.Settings {
		switch s.Key {
		case "vcs.revision":
			ret.MainRev = s.Value
			// vcs.modified not needed, go will include the +dirty for us
		}
	}

	main := bi.Main
	ret.MainModule = main.Path
	ret.MainVersion = main.Version
	if rev := ret.MainRev; rev != "" {
		if len(rev) > 8 {
			rev = rev[:8]
		}
		// go revisions often contain the git hash already
		if !strings.Contains(ret.MainVersion, rev) {
			ret.MainVersion += "+" + rev
		}
	}

	const gdevPath = "fastcat.org/go/gdev"
	if main.Path == gdevPath {
		ret.GDevVersion = ret.MainVersion
	} else {
		for _, m := range bi.Deps {
			if m.Path == gdevPath {
				ret.GDevVersion = m.Version
				break
			}
		}
	}

	return ret
}
