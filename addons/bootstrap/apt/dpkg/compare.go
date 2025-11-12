package dpkgcomp

import (
	"errors"
	"strconv"
	"strings"
	"unicode"
)

// Version represents a Debian-like package version
type Version struct {
	Epoch           int
	UpstreamVersion string
	DebianRevision  string
}

var upstreamVersionAllowedSymbols = []rune{'.', '-', '+', '~', ':', '_'}
var debianRevisionAllowedSymbols = []rune{'.', '+', '~', '_'}

// Compare function compares two Debian-like package version
//
// The implementation is based on http://man.he.net/man5/deb-version
// on https://www.debian.org/doc/debian-policy/ch-controlfields.html#s-f-Version
//
// It uses the dpkg-1.17.25's algorithm  (lib/version.c)
func Compare(a, b string) (int, error) {
	v1, err := StringToVersion(a)
	if err != nil {
		return 0, err
	}

	v2, err := StringToVersion(b)
	if err != nil {
		return 0, err
	}

	return CompareV(v1, v2), nil
}

// CompareV function compares two Debian-like package version
//
// The implementation is based on http://man.he.net/man5/deb-version
// on https://www.debian.org/doc/debian-policy/ch-controlfields.html#s-f-Version
//
// It uses the dpkg-1.17.25's algorithm  (lib/version.c)
func CompareV(a, b Version) int {
	// Quick check
	if a == b {
		return 0
	}

	// Compare epochs
	if a.Epoch > b.Epoch {
		return 1
	}
	if a.Epoch < b.Epoch {
		return -1
	}

	// Compare UpstreamVersion
	rc := verrevcmp(a.UpstreamVersion, b.UpstreamVersion)
	if rc != 0 {
		return signum(rc)
	}

	// Compare DebianRevision
	return signum(verrevcmp(a.DebianRevision, b.DebianRevision))
}

// StringToVersion function parses a string into a Version struct which can be compared
//
// The implementation is based on http://man.he.net/man5/deb-version
// on https://www.debian.org/doc/debian-policy/ch-controlfields.html#s-f-Version
//
// It uses the dpkg-1.17.25's algorithm  (lib/parsehelp.c)
func StringToVersion(str string) (Version, error) {
	var version Version

	// Trim leading and trailing space
	str = strings.TrimSpace(str)

	if len(str) <= 0 {
		return Version{}, errors.New("Version string is empty")
	}

	// Find Epoch
	sepEpoch := strings.Index(str, ":")
	if sepEpoch > -1 {
		intEpoch, err := strconv.Atoi(str[:sepEpoch])
		if err == nil {
			version.Epoch = intEpoch
		} else {
			return Version{}, errors.New("Epoch in version is not a number")
		}
		if intEpoch < 0 {
			return Version{}, errors.New("Epoch in version is negative")
		}
	} else {
		version.Epoch = 0
	}

	// Find UpstreamVersion / DebianRevision
	sepDebianRevision := strings.LastIndex(str, "-")
	if sepDebianRevision > -1 {
		version.UpstreamVersion = str[sepEpoch+1 : sepDebianRevision]
		version.DebianRevision = str[sepDebianRevision+1:]
	} else {
		version.UpstreamVersion = str[sepEpoch+1:]
		version.DebianRevision = "0"
	}
	// Verify format
	if len(version.UpstreamVersion) == 0 {
		return Version{}, errors.New("No UpstreamVersion in version")
	}

	if !unicode.IsDigit(rune(version.UpstreamVersion[0])) {
		return Version{}, errors.New("UpstreamVersion in version does not start with digit")
	}

	for i := 0; i < len(version.UpstreamVersion); i = i + 1 {
		r := rune(version.UpstreamVersion[i])
		if !unicode.IsDigit(r) && !unicode.IsLetter(r) && !containsRune(upstreamVersionAllowedSymbols, r) {
			return Version{}, errors.New("invalid character in UpstreamVersion")
		}
	}

	for i := 0; i < len(version.DebianRevision); i = i + 1 {
		r := rune(version.DebianRevision[i])
		if !unicode.IsDigit(r) && !unicode.IsLetter(r) && !containsRune(debianRevisionAllowedSymbols, r) {
			return Version{}, errors.New("invalid character in DebianRevision")
		}
	}

	return version, nil
}

func verrevcmp(t1, t2 string) int {
	t1, rt1 := nextRune(t1)
	t2, rt2 := nextRune(t2)

	for rt1 != nil || rt2 != nil {
		firstDiff := 0

		for (rt1 != nil && !unicode.IsDigit(*rt1)) || (rt2 != nil && !unicode.IsDigit(*rt2)) {
			ac := 0
			bc := 0
			if rt1 != nil {
				ac = order(*rt1)
			}
			if rt2 != nil {
				bc = order(*rt2)
			}

			if ac != bc {
				return ac - bc
			}

			t1, rt1 = nextRune(t1)
			t2, rt2 = nextRune(t2)
		}
		for rt1 != nil && *rt1 == '0' {
			t1, rt1 = nextRune(t1)
		}
		for rt2 != nil && *rt2 == '0' {
			t2, rt2 = nextRune(t2)
		}
		for rt1 != nil && unicode.IsDigit(*rt1) && rt2 != nil && unicode.IsDigit(*rt2) {
			if firstDiff == 0 {
				firstDiff = int(*rt1) - int(*rt2)
			}
			t1, rt1 = nextRune(t1)
			t2, rt2 = nextRune(t2)
		}
		if rt1 != nil && unicode.IsDigit(*rt1) {
			return 1
		}
		if rt2 != nil && unicode.IsDigit(*rt2) {
			return -1
		}
		if firstDiff != 0 {
			return firstDiff
		}
	}

	return 0
}

// order compares runes using a modified ASCII table
// so that letters are sorted earlier than non-letters
// and so that tildes sorts before anything
func order(r rune) int {
	if unicode.IsDigit(r) {
		return 0
	}

	if unicode.IsLetter(r) {
		return int(r)
	}

	if r == '~' {
		return -1
	}

	return int(r) + 256
}

func nextRune(str string) (string, *rune) {
	if len(str) >= 1 {
		r := rune(str[0])
		return str[1:], &r
	}
	return str, nil
}

func containsRune(s []rune, e rune) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func signum(a int) int {
	switch {
	case a < 0:
		return -1
	case a > 0:
		return +1
	}

	return 0
}
