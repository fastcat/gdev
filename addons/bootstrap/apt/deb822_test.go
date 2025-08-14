package apt

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatDeb822(t *testing.T) {
	tests := []struct {
		name    string
		content map[string]string
		want    []string
	}{
		{
			"trixie with source",
			map[string]string{
				"Types":      "deb deb-src",
				"URIs":       "https://deb.debian.org/debian/",
				"Suites":     "trixie",
				"Components": "main non-free non-free-firmware contrib",
				"Signed-By":  "/usr/share/keyrings/debian-archive-keyring.gpg",
			},
			[]string{
				"Types: deb deb-src",
				"URIs: https://deb.debian.org/debian/",
				"Suites: trixie",
				"Components: main non-free non-free-firmware contrib",
				"Signed-By: /usr/share/keyrings/debian-archive-keyring.gpg",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := &bytes.Buffer{}
			require.NoError(t, FormatDeb822Stanza(tt.content, deb822SourcesFirstKeys, got))
			gotLines := strings.Split(strings.TrimSpace(got.String()), "\n")
			assert.ElementsMatch(t, tt.want, gotLines)
		})
	}
}
