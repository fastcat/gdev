package apt_common

import (
	_ "embed"

	"fastcat.org/go/gdev/addons/bootstrap/apt"
)

//go:embed github-cli.asc
var GitHubCliArchiveKeyring []byte

func GitHubCLIInstaller() *apt.SourceInstaller {
	return &apt.SourceInstaller{
		SourceName: "github-cli",
		Source: &apt.Source{
			Types:      []string{"deb"},
			URIs:       []string{"https://cli.github.com/packages"},
			Suites:     []string{"stable"},
			Components: []string{"main"},
			SignedBy:   "/usr/share/keyrings/githubcli-archive-keyring.asc",
		},
		SigningKey: GitHubCliArchiveKeyring,
		// prefer the modernized format since GH doesn't ship anything specific in the package
		Deb822: true,
	}
}
