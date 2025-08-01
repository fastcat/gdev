package apt

import (
	"bytes"
	"errors"
	"io"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gotest.tools/v3/assert"

	"fastcat.org/go/gdev/shx"
)

// debian bookworm keyring from July 2025
var armoredLines = []string{
	`-----BEGIN PGP PUBLIC KEY BLOCK-----`,
	``,
	`mDMEY865UxYJKwYBBAHaRw8BAQdAd7Z0srwuhlB6JKFkcf4HU4SSS/xcRfwEQWzr`,
	`crf6AEq0SURlYmlhbiBTdGFibGUgUmVsZWFzZSBLZXkgKDEyL2Jvb2t3b3JtKSA8`,
	`ZGViaWFuLXJlbGVhc2VAbGlzdHMuZGViaWFuLm9yZz6IlgQTFggAPhYhBE1k/sEZ`,
	`wgKQZ9bnkfjSWFuHg9SBBQJjzrlTAhsDBQkPCZwABQsJCAcCBhUKCQgLAgQWAgMB`,
	`Ah4BAheAAAoJEPjSWFuHg9SBSgwBAP9qpeO5z1s5m4D4z3TcqDo1wez6DNya27QW`,
	`WoG/4oBsAQCEN8Z00DXagPHbwrvsY2t9BCsT+PgnSn9biobwX7bDDg==`,
	`=5NZE`,
	`-----END PGP PUBLIC KEY BLOCK-----`,
	``,
}

func TestAscToGPG(t *testing.T) {
	armored := strings.Join(armoredLines, "\n")
	var got bytes.Buffer
	require.NoError(t, AscToGPG(strings.NewReader(armored), &got))

	res, err := shx.Run(t.Context(),
		[]string{"gpg", "--dearmor"},
		shx.PassStderr(),
		shx.FeedStdin(strings.NewReader(armored)),
		shx.CaptureOutput(),
	)
	if errors.Is(err, exec.ErrNotFound) {
		t.Skip("gpg not found, skipping test")
	}
	require.NoError(t, err)
	require.NoError(t, res.Err())
	var want bytes.Buffer
	_, err = io.Copy(&want, res.Stdout())
	require.NoError(t, err)

	assert.DeepEqual(t, got.Bytes(), want.Bytes())
}
