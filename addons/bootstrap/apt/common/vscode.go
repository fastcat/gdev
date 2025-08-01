package apt_common

import (
	"bytes"
	_ "embed"
	"sync"

	"fastcat.org/go/gdev/addons/bootstrap/apt"
)

// VSCode uses the binary format, but we keep the text one in the source tree to
// keep things more human observable however, and convert it to binary on demand.
//
// See: [VSCodeArchiveKeyringBinary].
//
//go:embed vscode.asc
var VSCodeArchiveKeyring []byte

// Once generator for the binary version of the VSCode archive keyring.
//
// See [VSCodeArchiveKeyring].
var VSCodeArchiveKeyringBinary = sync.OnceValue(func() []byte {
	var buf bytes.Buffer
	if err := apt.AscToGPG(bytes.NewReader(VSCodeArchiveKeyring), &buf); err != nil {
		panic(err) // should never happen
	}
	return buf.Bytes()
})

func VSCodeInstaller() *apt.SourceInstaller {
	return &apt.SourceInstaller{
		SourceName: "vscode",
		Source: &apt.Source{
			Types:      []string{"deb"},
			URIs:       []string{"https://packages.microsoft.com/repos/vscode"},
			Suites:     []string{"stable"},
			Components: []string{"main"},
			SignedBy:   "/usr/share/keyrings/microsoft.gpg",
		},
		SigningKey: VSCodeArchiveKeyringBinary(),
		Deb822:     true,
	}
}
