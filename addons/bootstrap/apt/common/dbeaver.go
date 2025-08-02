package apt_common

import (
	_ "embed"

	"fastcat.org/go/gdev/addons/bootstrap/apt"
)

// use a fragment so it knows the format.
//
//go:generate go tool getkey https://dbeaver.io/debs/dbeaver.gpg.key#.asc dbeaver.asc
//go:embed dbeaver.asc
var DBeaverArchiveKeyring []byte

func DBeaverInstaller() *apt.SourceInstaller {
	return &apt.SourceInstaller{
		SourceName: "dbeaver",
		Source: &apt.Source{
			Types: []string{"deb"},
			URIs:  []string{"https://dbeaver.io/debs/dbeaver-ce"},
			// weird old repo setup
			Suites:     []string{"/"},
			Components: []string{""},
			SignedBy:   "/usr/share/keyrings/dbeaver.asc",
		},
		SigningKey: DBeaverArchiveKeyring,
		Deb822:     true,
	}
}
