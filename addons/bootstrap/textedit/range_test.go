package textedit

import (
	"testing"
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
			testEditor(
				t,
				tt.original,
				SpliceRange(tt.edit...),
				tt.expected,
				tt.expectSkip,
			)
		})
	}
}
