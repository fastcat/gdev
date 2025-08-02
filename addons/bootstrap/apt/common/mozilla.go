package apt_common

import (
	_ "embed"

	"fastcat.org/go/gdev/addons/bootstrap/apt"
)

// use a fragment to trick the code because while the url says .gpg, it returns an .asc file
//
//go:generate go tool getkey https://packages.mozilla.org/apt/repo-signing-key.gpg#.asc mozilla.asc
//go:embed mozilla.asc
var MozillaArchiveKeyring []byte

func MozillaInstaller() *apt.SourceInstaller {
	return &apt.SourceInstaller{
		SourceName: "mozilla",
		Source: &apt.Source{
			Types:      []string{"deb"},
			URIs:       []string{"https://packages.mozilla.org/apt"},
			Suites:     []string{"mozilla"},
			Components: []string{"main"},
			SignedBy:   "/usr/share/keyrings/packages.mozilla.org.asc",
		},
		SigningKey: MozillaArchiveKeyring,
		Deb822:     true,
	}
}
