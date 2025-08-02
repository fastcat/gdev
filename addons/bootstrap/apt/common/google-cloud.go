package apt_common

import (
	_ "embed"

	"fastcat.org/go/gdev/addons/bootstrap/apt"
)

// use a fragment to trick the code because while the url says .gpg, it returns an .asc file
//
//go:generate go tool getkey https://packages.cloud.google.com/apt/doc/apt-key.gpg#.asc google-cloud.asc
//go:embed google-cloud.asc
var GoogleCloudArchiveKeyring []byte

func GoogleCloudInstaller() *apt.SourceInstaller {
	return &apt.SourceInstaller{
		SourceName: "google-cloud-sdk",
		Source: &apt.Source{
			Types:      []string{"deb"},
			URIs:       []string{"https://packages.cloud.google.com/apt"},
			Suites:     []string{"cloud-sdk"},
			Components: []string{"main"},
			SignedBy:   "/usr/share/keyrings/cloud.google.asc",
		},
		SigningKey: GoogleCloudArchiveKeyring,
		Deb822:     true,
	}
}
