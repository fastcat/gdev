package gocache_http

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_backend_FullName(t *testing.T) {
	tests := []struct {
		name string
		base string
		path string
		want string
	}{
		{
			"server root + plain name",
			"http://example.com",
			"file.txt",
			"http://example.com/file.txt",
		},
		{
			"server root with slash + plain name",
			"http://example.com/",
			"file.txt",
			"http://example.com/file.txt",
		},
		{
			"server root + path with slash",
			"http://example.com",
			"subdir/file.txt",
			"http://example.com/subdir/file.txt",
		},
		{
			"server root + bogus absolute path",
			"http://example.com/",
			"/subdir/file.txt",
			"http://example.com/subdir/file.txt",
		},
		{
			"server subdir + plain name",
			"http://example.com/subdir",
			"file.txt",
			"http://example.com/subdir/file.txt",
		},
		{
			"server subdir + path with slash",
			"http://example.com/subdir",
			"subdir/file.txt",
			"http://example.com/subdir/subdir/file.txt",
		},
		{
			"server subdir + bogus absolute path",
			"http://example.com/subdir1",
			"/subdir2/file.txt",
			"http://example.com/subdir1/subdir2/file.txt",
		},
		{
			"server subdir with slash + deep path",
			"http://example.com/subdir/",
			"subdir2/subdir3/file.txt",
			"http://example.com/subdir/subdir2/subdir3/file.txt",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			be, err := newBackend(nil, nil, tt.base)
			require.NoError(t, err)
			assert.Equal(t, tt.want, be.FullName(tt.path))
		})
	}
}
