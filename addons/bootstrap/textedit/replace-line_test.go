package textedit

import (
	"testing"
)

func TestReplaceLine(t *testing.T) {
	type test struct {
		name       string
		original   []string
		oldLine    string
		newLine    string
		prevLines  []string
		expected   []string
		expectSkip bool
	}
	tests := []test{
		{
			name:       "skip empty file",
			original:   []string{},
			oldLine:    "old",
			newLine:    "new",
			expectSkip: true,
		},
		{
			name:     "simple",
			original: []string{"old\n"},
			oldLine:  "old",
			newLine:  "new",
			expected: []string{"new\n"},
		},
		{
			name:      "with previous lines",
			original:  []string{"ctx1\n", "pfx1\n", "pfx2\n", "old\n", "ctx2\n"},
			oldLine:   "old",
			newLine:   "new",
			prevLines: []string{"pfx1", "pfx2"},
			expected:  []string{"ctx1\n", "pfx1\n", "pfx2\n", "new\n", "ctx2\n"},
		},
		{
			name:      "with previous lines and partial match 1",
			original:  []string{"ctx1\n", "pfx1\n", "ctx2\n", "pfx1\n", "pfx2\n", "old\n", "ctx3\n"},
			oldLine:   "old",
			newLine:   "new",
			prevLines: []string{"pfx1", "pfx2"},
			expected:  []string{"ctx1\n", "pfx1\n", "ctx2\n", "pfx1\n", "pfx2\n", "new\n", "ctx3\n"},
		},
		{
			name:       "with previous lines and partial match 2",
			original:   []string{"ctx1\n", "pfx1\n", "ctx2\n", "pfx2\n", "old\n", "ctx3\n"},
			oldLine:    "old",
			newLine:    "new",
			prevLines:  []string{"pfx1", "pfx2"},
			expectSkip: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			testEditor(
				t,
				tt.original,
				ReplaceLine(tt.oldLine, tt.newLine).WithPreviousLines(tt.prevLines...),
				tt.expected,
				tt.expectSkip,
			)
		})
	}
}
