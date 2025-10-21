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

func testEditor(
	t testing.TB,
	original []string,
	editor Editor,
	expected []string,
	expectSkip bool,
) {
	fn := filepath.Join(t.TempDir(), "test.txt")
	require.NoError(t, os.WriteFile(fn, []byte(strings.Join(original, "")), 0o644))
	st1, err := os.Stat(fn)
	require.NoError(t, err)
	// ensure system clock moves forwards so file mtime change detection
	// works. For whatever reason filesystem timestamps are sometimes a couple
	// ms in the past.
	time.Sleep(4 * time.Millisecond)

	changed, err := EditFile(fn, editor)
	require.NoError(t, err)
	assert.Equal(t, expectSkip, !changed)

	got, err := os.ReadFile(fn)
	require.NoError(t, err)
	st2, err := os.Stat(fn)
	require.NoError(t, err)

	if expectSkip {
		assert.Equal(t, strings.Join(original, ""), string(got))
		assert.Equal(t, st1.ModTime(), st2.ModTime(), "file mtime should not have changed")
		assert.Equal(t, inode(st1), inode(st2), "inode should not have changed")
	} else {
		assert.Equal(t, strings.Join(expected, ""), string(got))
		assert.Greater(t, st2.ModTime(), st1.ModTime(), "file mtime should have increased")
		assert.NotEqual(t, inode(st1), inode(st2), "inode should have changed")
	}
}
