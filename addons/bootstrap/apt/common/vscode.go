package apt_common

import (
	"bytes"
	_ "embed"
	"io"
	"sync"

	"golang.org/x/crypto/openpgp/armor" //nolint:staticcheck // armor parsing is fine within deprecation

	"fastcat.org/go/gdev/addons/bootstrap/apt"
)

// VSCode uses the binary format, but we keep the text one in the source tree to
// keep things more human observable however, and convert it to binary on demand.
//
// See: [VSCodeArchiveKeyringBinary].
//
//go:generate go tool getkey https://packages.microsoft.com/keys/microsoft.asc vscode.asc
//go:embed vscode.asc
var VSCodeArchiveKeyring []byte

// Once generator for the binary version of the VSCode archive keyring.
//
// See [VSCodeArchiveKeyring].
var VSCodeArchiveKeyringBinary = sync.OnceValue(func() []byte {
	block, err := armor.Decode(bytes.NewReader(VSCodeArchiveKeyring))
	if err != nil {
		panic(err) // should never happen
	}
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, block.Body); err != nil {
		panic(err) // should never happen
	}
	return buf.Bytes()
})

func VSCodeInstaller() *apt.SourceInstaller {
	return &apt.SourceInstaller{
		SourceName: "vscode",
		Source: &apt.Source{
			Types:         []string{"deb"},
			URIs:          []string{"https://packages.microsoft.com/repos/code"},
			Suites:        []string{"stable"},
			Components:    []string{"main"},
			Architectures: []string{"amd64", "arm64", "armhf"},
			SignedBy:      "/usr/share/keyrings/microsoft.gpg",
		},
		SigningKey: VSCodeArchiveKeyringBinary(),
		Deb822:     true,
	}
}
