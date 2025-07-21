package textedit

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAppendLine(t *testing.T) {
	type test struct {
		name       string
		original   []string
		line       string
		oldLines   []string
		expected   []string
		expectSkip bool
	}
	tests := []test{
		{
			name:     "append to empty file",
			line:     "new line",
			expected: []string{"new line\n"},
		},
		{
			name:     "append to non-empty file",
			original: []string{"line 1\n", "line 2\n"},
			line:     "new line",
			expected: []string{"line 1\n", "line 2\n", "new line\n"},
		},
		{
			name:       "no-op already exists",
			original:   []string{"line 1\n", "new line\n", "line 2\n"},
			line:       "new line",
			expectSkip: true,
		},
		{
			name:     "replace old line",
			original: []string{"line 1\n", "old line\n", "line 2\n"},
			line:     "new line",
			oldLines: []string{"old line"},
			expected: []string{"line 1\n", "new line\n", "line 2\n"},
		},
		{
			name:       "retain existing whitespace",
			original:   []string{"line 1\n", "\tnew line\n", "line 2\n"},
			line:       "new line",
			expectSkip: true,
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

			editor := AppendLine(tt.line, tt.oldLines...)
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

func inode(st os.FileInfo) uint64 {
	stt := st.Sys().(*syscall.Stat_t)
	return stt.Ino
}
