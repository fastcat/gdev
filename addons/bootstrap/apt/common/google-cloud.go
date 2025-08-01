package apt_common

import (
	_ "embed"

	"fastcat.org/go/gdev/addons/bootstrap/apt"
)

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
