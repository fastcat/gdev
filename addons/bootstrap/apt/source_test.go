package apt

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestAptSource_ToList(t *testing.T) {
	tests := []struct {
		name   string
		source AptSource
		want   string
	}{
		{
			"bookworm with source",
			AptSource{
				Types:      []string{"deb", "deb-src"},
				URIs:       []string{"https://deb.debian.org/debian/"},
				Suites:     []string{"bookworm"},
				Components: []string{"main", "non-free", "non-free-firmware", "contrib"},
				SignedBy:   "/usr/share/keyrings/debian-archive-keyring.gpg",
			},
			"deb [signed-by=/usr/share/keyrings/debian-archive-keyring.gpg] https://deb.debian.org/debian/ bookworm main non-free non-free-firmware contrib\n" +
				"deb-src [signed-by=/usr/share/keyrings/debian-archive-keyring.gpg] https://deb.debian.org/debian/ bookworm main non-free non-free-firmware contrib\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, string(tt.source.ToList()))
		})
	}
}

func TestAptSource_ToDeb822(t *testing.T) {
	tests := []struct {
		name   string
		source AptSource
		want   map[string]string
	}{
		{
			"trixie with source",
			AptSource{
				Types:      []string{"deb", "deb-src"},
				URIs:       []string{"https://deb.debian.org/debian/"},
				Suites:     []string{"trixie"},
				Components: []string{"main", "non-free", "non-free-firmware", "contrib"},
				SignedBy:   "/usr/share/keyrings/debian-archive-keyring.gpg",
			},
			map[string]string{
				"Types":      "deb deb-src",
				"URIs":       "https://deb.debian.org/debian/",
				"Suites":     "trixie",
				"Components": "main non-free non-free-firmware contrib",
				"Signed-By":  "/usr/share/keyrings/debian-archive-keyring.gpg",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.DeepEqual(t, tt.want, tt.source.ToDeb822())
		})
	}
}
