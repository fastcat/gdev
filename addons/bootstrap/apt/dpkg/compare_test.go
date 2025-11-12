package dpkgcomp

import (
	"strings"
	"testing"
)

//TODO Refactor me ... I do my job but stewpidly ...

func TestCompareV(t *testing.T) {
	var a, b Version

	// Test for blank version equality
	a = Version{}
	b = Version{}
	if cmp := CompareV(a, b); cmp != 0 {
		t.Error("Blank comparison failed")
	}

	// Preliminary tests
	a = Version{Epoch: 1}
	b = Version{Epoch: 2}
	if cmp := CompareV(a, b); cmp == 0 {
		t.Error("Epoch comparison failed")
	}

	a = Version{Epoch: 0, UpstreamVersion: "1", DebianRevision: "1"}
	b = Version{Epoch: 2, UpstreamVersion: "2", DebianRevision: "1"}
	if cmp := CompareV(a, b); cmp == 0 {
		t.Error("UpstreamVersion comparison failed")
	}

	a = Version{Epoch: 0, UpstreamVersion: "1", DebianRevision: "1"}
	b = Version{Epoch: 2, UpstreamVersion: "1", DebianRevision: "2"}
	if cmp := CompareV(a, b); cmp == 0 {
		t.Error("DebianRevision comparison failed")
	}

	// Test for version equality
	a = Version{Epoch: 0, UpstreamVersion: "0", DebianRevision: "0"}
	b = Version{Epoch: 0, UpstreamVersion: "0", DebianRevision: "0"}
	if cmp := CompareV(a, b); cmp != 0 {
		t.Error("Equality comparison failed")
	}

	a = Version{Epoch: 0, UpstreamVersion: "0", DebianRevision: "00"}
	b = Version{Epoch: 0, UpstreamVersion: "00", DebianRevision: "0"}
	if cmp := CompareV(a, b); cmp != 0 {
		t.Error("Equality comparison failed")
	}

	a = Version{Epoch: 1, UpstreamVersion: "2", DebianRevision: "3"}
	b = Version{Epoch: 1, UpstreamVersion: "2", DebianRevision: "3"}
	if cmp := CompareV(a, b); cmp != 0 {
		t.Error("Equality comparison failed")
	}

	// Test for epoch difference
	a = Version{Epoch: 0, UpstreamVersion: "0", DebianRevision: "0"}
	b = Version{Epoch: 1, UpstreamVersion: "0", DebianRevision: "0"}
	if cmp := CompareV(a, b); cmp > 0 {
		t.Error("Epoch comparison failed")
	}
	if cmp := CompareV(b, a); cmp < 0 {
		t.Error("Epoch comparison failed")
	}

	// Test for UpstreamVersion component difference
	a = Version{Epoch: 0, UpstreamVersion: "a", DebianRevision: "0"}
	b = Version{Epoch: 0, UpstreamVersion: "b", DebianRevision: "0"}
	if cmp := CompareV(a, b); cmp > 0 {
		t.Error("UpstreamVersion comparison failed")
	}
	if cmp := CompareV(b, a); cmp < 0 {
		t.Error("UpstreamVersion comparison failed")
	}

	// Test for DebianRevision component difference
	a = Version{Epoch: 0, UpstreamVersion: "0", DebianRevision: "a"}
	b = Version{Epoch: 0, UpstreamVersion: "0", DebianRevision: "b"}
	if cmp := CompareV(a, b); cmp > 0 {
		t.Error("DebianRevision comparison failed")
	}
	if cmp := CompareV(b, a); cmp < 0 {
		t.Error("DebianRevision comparison failed")
	}

	//FIXME Complete me
}

func TestParse(t *testing.T) {
	var a, b Version
	var err error

	// Test 0 versions
	b = Version{Epoch: 0, UpstreamVersion: "0", DebianRevision: ""}

	a, _ = StringToVersion("0")
	if cmp := CompareV(a, b); cmp != 0 {
		t.Error("Parse blank failed")
	}

	a, _ = StringToVersion("0:0")
	if cmp := CompareV(a, b); cmp != 0 {
		t.Error("Parse blank failed")
	}

	a, _ = StringToVersion("0:0-")
	if cmp := CompareV(a, b); cmp != 0 {
		t.Error("Parse blank failed")
	}

	a, _ = StringToVersion("0:0-0")
	b = Version{Epoch: 0, UpstreamVersion: "0", DebianRevision: "0"}
	if cmp := CompareV(a, b); cmp != 0 {
		t.Error("Parse blank failed")
	}

	a, _ = StringToVersion("0:0.0-0.0")
	b = Version{Epoch: 0, UpstreamVersion: "0.0", DebianRevision: "0.0"}
	if cmp := CompareV(a, b); cmp != 0 {
		t.Error("Parse blank failed")
	}

	// Test epoched versions
	a, _ = StringToVersion("1:0")
	b = Version{Epoch: 1, UpstreamVersion: "0", DebianRevision: ""}
	if cmp := CompareV(a, b); cmp != 0 {
		t.Error("Parse epoched failed")
	}

	a, _ = StringToVersion("5:1")
	b = Version{Epoch: 5, UpstreamVersion: "1", DebianRevision: ""}
	if cmp := CompareV(a, b); cmp != 0 {
		t.Error("Parse epoched failed")
	}

	// Test multiple hyphens
	a, _ = StringToVersion("0:0-0-0")
	b = Version{Epoch: 0, UpstreamVersion: "0-0", DebianRevision: "0"}
	if cmp := CompareV(a, b); cmp != 0 {
		t.Error("Parse multiple hyphens failed")
	}

	a, _ = StringToVersion("0:0-0-0-0")
	b = Version{Epoch: 0, UpstreamVersion: "0-0-0", DebianRevision: ""}
	if cmp := CompareV(a, b); cmp != 0 {
		t.Error("Parse multiple hyphens failed")
	}

	// Test multiple colons
	a, _ = StringToVersion("0:0:0-0")
	b = Version{Epoch: 0, UpstreamVersion: "0:0", DebianRevision: "0"}
	if cmp := CompareV(a, b); cmp != 0 {
		t.Error("Parse multiple colons failed")
	}

	a, _ = StringToVersion("0:0:0:0-0")
	b = Version{Epoch: 0, UpstreamVersion: "0:0:0", DebianRevision: "0"}
	if cmp := CompareV(a, b); cmp != 0 {
		t.Error("Parse multiple colons failed")
	}

	// Test multiple hyphens and colons
	a, _ = StringToVersion("0:0:0-0-0")
	b = Version{Epoch: 0, UpstreamVersion: "0:0-0", DebianRevision: "0"}
	if cmp := CompareV(a, b); cmp != 0 {
		t.Error("Parse multiple hyphens and colons failed")
	}

	a, _ = StringToVersion("0:0-0:0-0")
	b = Version{Epoch: 0, UpstreamVersion: "0-0:0", DebianRevision: "0"}
	if cmp := CompareV(a, b); cmp != 0 {
		t.Error("Parse multiple hyphens and colons failed")
	}

	// Test valid characters in upstream version
	a, _ = StringToVersion("0:09azAZ.-+~:-0")
	b = Version{Epoch: 0, UpstreamVersion: "09azAZ.-+~:", DebianRevision: "0"}
	if cmp := CompareV(a, b); cmp != 0 {
		t.Error("Parse valid characters in UpstreamVersion failed")
	}

	// Test valid characters in debian revision
	a, _ = StringToVersion("0:0-azAZ09.+~")
	b = Version{Epoch: 0, UpstreamVersion: "0", DebianRevision: "azAZ09.+~"}
	if cmp := CompareV(a, b); cmp != 0 {
		t.Error("Parse valid characters in DebianRevision failed")
	}

	// Test version with leading and trailing spaces
	a, _ = StringToVersion("  	0:0-1")
	b = Version{Epoch: 0, UpstreamVersion: "0", DebianRevision: "1"}
	if cmp := CompareV(a, b); cmp != 0 {
		t.Error("Parse with leading and trailing spaces failed")
	}

	a, _ = StringToVersion("0:0-1	  ")
	b = Version{Epoch: 0, UpstreamVersion: "0", DebianRevision: "1"}
	if cmp := CompareV(a, b); cmp != 0 {
		t.Error("Parse with leading and trailing spaces failed")
	}

	a, _ = StringToVersion("	  0:0-1  	")
	b = Version{Epoch: 0, UpstreamVersion: "0", DebianRevision: "1"}
	if cmp := CompareV(a, b); cmp != 0 {
		t.Error("Parse with leading and trailing spaces failed")
	}

	// Test empty version
	if _, err = StringToVersion(""); err == nil {
		t.Error("Parse with empty version failed")
	}

	if _, err = StringToVersion("  "); err == nil {
		t.Error("Parse with empty version failed")
	}

	if _, err = StringToVersion("0:"); err == nil {
		t.Error("Parse with empty upstream version failed")
	}

	// Test version with embedded spaces
	if _, err = StringToVersion("0:0 0-1"); err == nil {
		t.Error("Parse with embedded spaces failed")
	}

	// Test version with negative Epoch
	if _, err = StringToVersion("-1:0-1"); err == nil {
		t.Error("Parse with negative Epoch failed")
	}

	// Test invalid characters in Epoch
	if _, err = StringToVersion("a:0-0"); err == nil {
		t.Error("Parse with negative Epoch failed")
	}

	if _, err = StringToVersion("A:0-0"); err == nil {
		t.Error("Parse with negative Epoch failed")
	}

	// Test upstream version not starting with a digit
	if _, err = StringToVersion("0:abc3-0"); err == nil {
		t.Error("Parse with UpstreamVersion not starting with a digit failed")
	}

	// Test invalid characters in upstream version.
	versym := []rune{'!', '#', '@', '$', '%', '&', '/', '|', '\\', '<', '>', '(', ')', '[', ']', '{', '}', ';', ',', '_', '=', '*', '^', '\''}
	for _, r := range versym {
		verstr := strings.Join([]string{"0:0", string(r), "-0"}, "")
		if _, err = StringToVersion(verstr); err == nil {
			t.Error("Parse with invalid characters in UpstreamVersion failed")
		}
	}

	// Test invalid characters in revision
	if _, err = StringToVersion("0:0-0:0"); err == nil {
		t.Error("Parse with invalid characters in DebianRevision failed")
	}

	versym = []rune{'!', '#', '@', '$', '%', '&', '/', '|', '\\', '<', '>', '(', ')', '[', ']', '{', '}', ':', ';', ',', '=', '*', '^', '\''}
	for _, r := range versym {
		verstr := strings.Join([]string{"0:0-", string(r)}, "")
		if _, err = StringToVersion(verstr); err == nil {
			t.Error("Parse with invalid characters in DebianRevision failed")
		}
	}
}

func TestCompare(t *testing.T) {
	const LESS = -1
	const EQUAL = 0
	const GREATER = 1

	cases := []struct {
		v1     string
		expect int
		v2     string
	}{
		{"7.6p2-4", GREATER, "7.6-0"},
		{"1.0.3-3", GREATER, "1.0-1"},
		{"1.3", GREATER, "1.2.2-2"},
		{"1.3", GREATER, "1.2.2"},
		// Some properties of text strings
		{"0-pre", EQUAL, "0-pre"},
		{"0-pre", LESS, "0-pree"},
		{"1.1.6r2-2", GREATER, "1.1.6r-1"},
		{"2.6b2-1", GREATER, "2.6b-2"},
		{"98.1p5-1", LESS, "98.1-pre2-b6-2"},
		{"0.4a6-2", GREATER, "0.4-1"},
		{"1:3.0.5-2", LESS, "1:3.0.5.1"},
		// Epochs
		{"1:0.4", GREATER, "10.3"},
		{"1:1.25-4", LESS, "1:1.25-8"},
		{"0:1.18.36", EQUAL, "1.18.36"},
		{"1.18.36", GREATER, "1.18.35"},
		{"0:1.18.36", GREATER, "1.18.35"},
		// Funky, but allowed, characters in upstream version
		{"9:1.18.36:5.4-20", LESS, "10:0.5.1-22"},
		{"9:1.18.36:5.4-20", LESS, "9:1.18.36:5.5-1"},
		{"9:1.18.36:5.4-20", LESS, " 9:1.18.37:4.3-22"},
		{"1.18.36-0.17.35-18", GREATER, "1.18.36-19"},
		// Junk
		{"1:1.2.13-3", LESS, "1:1.2.13-3.1"},
		{"2.0.7pre1-4", LESS, "2.0.7r-1"},
		// if a version includes a dash, it should be the debrev dash - policy says so
		{"0:0-0-0", GREATER, "0-0"},
		// do we like strange versions? Yes we like strange versions…
		{"0", EQUAL, "0"},
		{"0", EQUAL, "00"},
		// #205960
		{"3.0~rc1-1", LESS, "3.0-1"},
		// #573592 - debian policy 5.6.12
		{"1.0", EQUAL, "1.0-0"},
		{"0.2", LESS, "1.0-0"},
		{"1.0", LESS, "1.0-0+b1"},
		{"1.0", GREATER, "1.0-0~"},
		// "steal" the testcases from (old perl) cupt
		{"1.2.3", EQUAL, "1.2.3"},                           // identical
		{"4.4.3-2", EQUAL, "4.4.3-2"},                       // identical
		{"1:2ab:5", EQUAL, "1:2ab:5"},                       // this is correct...
		{"7:1-a:b-5", EQUAL, "7:1-a:b-5"},                   // and this
		{"57:1.2.3abYZ+~-4-5", EQUAL, "57:1.2.3abYZ+~-4-5"}, // and those too
		{"1.2.3", EQUAL, "0:1.2.3"},                         // zero epoch
		{"1.2.3", EQUAL, "1.2.3-0"},                         // zero revision
		{"009", EQUAL, "9"},                                 // zeroes…
		{"009ab5", EQUAL, "9ab5"},                           // there as well
		{"1.2.3", LESS, "1.2.3-1"},                          // added non-zero revision
		{"1.2.3", LESS, "1.2.4"},                            // just bigger
		{"1.2.4", GREATER, "1.2.3"},                         // order doesn't matter
		{"1.2.24", GREATER, "1.2.3"},                        // bigger, eh?
		{"0.10.0", GREATER, "0.8.7"},                        // bigger, eh?
		{"3.2", GREATER, "2.3"},                             // major number rocks
		{"1.3.2a", GREATER, "1.3.2"},                        // letters rock
		{"0.5.0~git", LESS, "0.5.0~git2"},                   // numbers rock
		{"2a", LESS, "21"},                                  // but not in all places
		{"1.3.2a", LESS, "1.3.2b"},                          // but there is another letter
		{"1:1.2.3", GREATER, "1.2.4"},                       // epoch rocks
		{"1:1.2.3", LESS, "1:1.2.4"},                        // bigger anyway
		{"1.2a+~bCd3", LESS, "1.2a++"},                      // tilde doesn't rock
		{"1.2a+~bCd3", GREATER, "1.2a+~"},                   // but first is longer!
		{"5:2", GREATER, "304-2"},                           // epoch rocks
		{"5:2", LESS, "304:2"},                              // so big epoch?
		{"25:2", GREATER, "3:2"},                            // 25 > 3, obviously
		{"1:2:123", LESS, "1:12:3"},                         // 12 > 2
		{"1.2-5", LESS, "1.2-3-5"},                          // 1.2 < 1.2-3
		{"5.10.0", GREATER, "5.005"},                        // preceding zeroes don't matters
		{"3a9.8", LESS, "3.10.2"},                           // letters are before all letter symbols
		{"3a9.8", GREATER, "3~10"},                          // but after the tilde
		{"1.4+OOo3.0.0~", LESS, "1.4+OOo3.0.0-4"},           // another tilde check
		{"2.4.7-1", LESS, "2.4.7-z"},                        // revision comparing
		{"1.002-1+b2", GREATER, "1.00"},                     // whatever...
	}

	for _, c := range cases {
		if cmp, err := Compare(c.v1, c.v2); cmp != c.expect {
			t.Error("Comparison test failed:", c.v1, "vs.", c.v2, "=", cmp, "while we expected", c.expect, "( Error:", err, ")")
		}
	}

	// Test invalid versions
	if _, err := Compare("%", "1"); err == nil {
		t.Error("Invalid version comparison test failed")
	}
	if _, err := Compare("1", "."); err == nil {
		t.Error("Invalid version comparison test failed")
	}
}
