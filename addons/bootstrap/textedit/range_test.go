package textedit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSpliceRange(t *testing.T) {
	type test struct {
		name       string
		original   []string
		edit       []string
		expected   []string
		expectSkip bool
	}
	tests := []test{
		{
			name:     "append to empty file",
			edit:     []string{"start", "content", "end"},
			expected: []string{"start\n", "content\n", "end\n"},
		},
		{
			name:     "append to non-empty file",
			original: []string{"line 1\n", "line 2\n"},
			edit:     []string{"start", "content", "end"},
			expected: []string{"line 1\n", "line 2\n", "start\n", "content\n", "end\n"},
		},
		{
			name:       "no-op already exists",
			original:   []string{"start\n", "content\n", "end\n"},
			edit:       []string{"start", "content", "end"},
			expectSkip: true,
		},
		{
			name:     "replace old range",
			original: []string{"line 1\n", "start\n", "old content\n", "end\n", "line 2\n"},
			edit:     []string{"start", "new content", "end"},
			expected: []string{"line 1\n", "start\n", "new content\n", "end\n", "line 2\n"},
		},
		// TODO: this editor doesn't do whitespace preservation
		{
			name:     "append with partial block at EOF",
			original: []string{"line 1\n", "start\n", "content\n"},
			edit:     []string{"start", "new content", "end"},
			expected: []string{"line 1\n", "start\n", "new content\n", "end\n"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			fn := filepath.Join(t.TempDir(), "test.txt")
			require.NoError(t, os.WriteFile(fn, []byte(strings.Join(tt.original, "")), 0o644))
			st1, err := os.Stat(fn)
			require.NoError(t, err)
			// ensure system clock moves forwards so file mtime change detection
			// works. For whatever reason filesystem timestamps are sometimes a couple
			// ms in the past.
			time.Sleep(4 * time.Millisecond)

			editor := SpliceRange(tt.edit...)
			require.NoError(t, EditFile(fn, editor))

			got, err := os.ReadFile(fn)
			require.NoError(t, err)
			st2, err := os.Stat(fn)
			require.NoError(t, err)

			if tt.expectSkip {
				assert.Equal(t, strings.Join(tt.original, ""), string(got))
				assert.Equal(t, st1.ModTime(), st2.ModTime(), "file mtime should not have changed")
				assert.Equal(t, inode(st1), inode(st2), "inode should not have changed")
			} else {
				assert.Equal(t, strings.Join(tt.expected, ""), string(got))
				assert.Greater(t, st2.ModTime(), st1.ModTime(), "file mtime should have increased")
				assert.NotEqual(t, inode(st1), inode(st2), "inode should have changed")
			}
		})
	}
}
