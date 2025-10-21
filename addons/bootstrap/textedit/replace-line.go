package textedit

import (
	"fmt"
	"iter"
	"strings"
)

// ReplaceLine makes an editor that will replace oldLine with newLine, with
// optional prior lines that must immediately precede old/new in order.
//
// This editor is stateful and can only be used once.
func ReplaceLine(oldLine, newLine string) *replaceLineEditor {
	oldLine = strings.TrimSpace(oldLine)
	newLine = strings.TrimSpace(newLine)
	if oldLine == "" || newLine == "" {
		panic(fmt.Errorf("ReplaceLine: old/new must not be empty after trimming whitespace"))
	}
	return &replaceLineEditor{
		oldLine: oldLine,
		newLine: newLine,
	}
}

type replaceLineEditor struct {
	oldLine, newLine string
	previousLines    []string
	mustMatch        bool
	// FUTURE: followingLines too

	found   bool
	prevPos int
}

func (r *replaceLineEditor) WithPreviousLines(previousLines ...string) *replaceLineEditor {
	for i := range previousLines {
		previousLines[i] = strings.TrimSpace(previousLines[i])
		if previousLines[i] == "" {
			panic(fmt.Errorf("ReplaceLine: previousLines must not contain empty lines after trimming whitespace"))
		}
	}
	r.previousLines = previousLines
	return r
}

// MustMatch makes the editor return an error if neither the old nor new state
// is found.
func (r *replaceLineEditor) MustMatch() *replaceLineEditor {
	r.mustMatch = true
	return r
}

// Found returns whether the replacement was made. It is only valid after the
// editor is used.
func (r *replaceLineEditor) Found() bool { return r.found }

// EOF implements Editor.
func (r *replaceLineEditor) EOF() (output iter.Seq[string], err error) {
	if r.mustMatch && !r.found {
		return nil, fmt.Errorf("ReplaceLine: neither old nor new line found")
	}
	return empty(), nil
}

// Next implements Editor.
func (r *replaceLineEditor) Next(line string) (output iter.Seq[string], err error) {
	if r.found {
		return each(line), nil
	}
	tsl := strings.TrimSpace(line)
	if r.prevPos >= len(r.previousLines) {
		if tsl == r.oldLine {
			r.found = true
			return each(r.newLine), nil
		}
	} else if tsl == r.previousLines[r.prevPos] {
		r.prevPos++
	} else {
		r.prevPos = 0
	}
	return each(line), nil
}
