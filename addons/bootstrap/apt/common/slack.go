package apt_common

import (
	_ "embed"

	"fastcat.org/go/gdev/addons/bootstrap/apt"
)

// use a fragment so it knows the format.
//
//go:generate go tool getkey https://packagecloud.io/slacktechnologies/slack/gpgkey#.asc slack.asc
//go:embed slack.asc
var SlackArchiveKeyring []byte

func SlackInstaller() *apt.SourceInstaller {
	return &apt.SourceInstaller{
		SourceName: "slack",
		Source: &apt.Source{
			Types: []string{"deb"},
			URIs:  []string{"https://packagecloud.io/slacktechnologies/slack/debian"},
			// not built for jessie, they just never updated the repo naming
			Suites:     []string{"jessie"},
			Components: []string{"main"},
			// TODO: upstream stores this in /etc/apt/trusted.gpg.d/slack-desktop.gpg
			SignedBy: "/usr/share/keyrings/slack-desktop.asc",
		},
		SigningKey: SlackArchiveKeyring,
		// upstream does a bad key install, we want to be sure to use the good one
		Deb822: true,
	}
}
