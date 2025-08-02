package apt_common

import (
	_ "embed"

	"fastcat.org/go/gdev/addons/bootstrap/apt"
)

//go:generate go tool getkey https://apt.releases.hashicorp.com/gpg hashicorp.asc
//go:embed hashicorp.asc
var HashicorpArchiveKeyring []byte

func HashicorpInstaller() *apt.SourceInstaller {
	return &apt.SourceInstaller{
		SourceName: "hashicorp",
		Source: &apt.Source{
			Types: []string{"deb"},
			URIs:  []string{"https://apt.releases.hashicorp.com"},
			Suites: []string{
				// hashicorp tends to lag in making new releases available
				HashicorpDistroFallback(
					HostOSVersionCodename(),
				),
			},
			Components: []string{"main"},
			SignedBy:   "/usr/share/keyrings/hashicorp.asc",
		},
		SigningKey: HashicorpArchiveKeyring,
		Deb822:     true,
	}
}

func HashicorpDistroFallback(codename string) string {
	switch codename {
	case "trixie":
		return "bookworm"
	default:
		return codename
	}
}
