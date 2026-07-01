package instance

import (
	"runtime/debug"
	"strings"
	"sync"
)

var info = sync.OnceValue(loadVersionInfo)

// versionOverride, if non-empty, replaces MainVersion from build info.
var versionOverride string

// SetVersion overrides the main version string reported by Version() and
// VersionInfo(). Call before any use of the version functions.
func SetVersion(v string) {
	versionOverride = v
}

func Version() string {
	if versionOverride != "" {
		return versionOverride
	}
	return info().MainVersion
}

func VersionInfo() versionInfo {
	vi := info()
	if versionOverride != "" {
		vi.MainVersion = versionOverride
	}
	return vi
}

type versionInfo struct {
	MainVersion string
	MainModule  string
	MainRev     string

	GDevVersion string

	IsDebugBuild bool
	CGOEnabled   bool
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
		case "-gcflags":
			// typically contains `all=-N -l`, possibly in the other order
			ret.IsDebugBuild = strings.Contains(s.Value, "-N") && strings.Contains(s.Value, "-l")
		case "CGO_ENABLED":
			// unsure if "true" ever appears
			ret.CGOEnabled = s.Value == "1" || s.Value == "true"
			// could do "cgo debug enabled" based on "CGO_CFLAGS"
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
