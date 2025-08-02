package apt_common

import (
	_ "embed"

	"fastcat.org/go/gdev/addons/bootstrap/apt"
)

//go:generate go tool getkey https://dl.google.com/linux/linux_signing_key.pub google-chrome.asc
//go:embed google-chrome.asc
var GoogleChromeArchiveKeyring []byte

func GoogleChromeInstaller() *apt.SourceInstaller {
	return &apt.SourceInstaller{
		SourceName: "google-chrome",
		Source: &apt.Source{
			Types:      []string{"deb"},
			URIs:       []string{"https://dl.google.com/linux/chrome/deb/"},
			Suites:     []string{"stable"},
			Components: []string{"main"},
			// max compat
			Architectures: []string{DpkgHostArchitecture()},
			// TODO: google stores this in /etc/apt/trusted.gpg.d/google-chrome.gpg
			// which makes it trusted for any source not just theirs
			SignedBy: "/usr/share/keyrings/google-chrome.asc",
		},
		SigningKey: GoogleChromeArchiveKeyring,
		// max compat with their setup
		Deb822: false,
	}
}
