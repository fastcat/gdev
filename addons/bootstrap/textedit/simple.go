package textedit

import (
	"fmt"
	"iter"
	"strings"
)

// AppendLine adds the given line to the end of the file if it is not already
// present. It will also insert it in place of any of the given oldLines
// (optional).
//
// It will panic if the line or any of the oldLines are empty after trimming
// leading & trailing whitespace.
func AppendLine(line string, oldLines ...string) Editor {
	lineTS := strings.TrimSpace(line)
	if lineTS == "" {
		panic(fmt.Errorf("AppendLine: line must not be empty after trimming whitespace"))
	}
	olSet := make(map[string]struct{}, len(oldLines))
	for _, oldLine := range oldLines {
		tsl := strings.TrimSpace(oldLine)
		if tsl == "" {
			panic(fmt.Errorf("AppendLine: oldLines must not contain empty lines after trimming whitespace"))
		}
		olSet[tsl] = struct{}{}
	}
	return &simpleEditor{line: line, lineTS: lineTS, oldLines: olSet}
}

type simpleEditor struct {
	line, lineTS string
	oldLines     map[string]struct{}

	found bool
}

// Next implements Editor.
func (s *simpleEditor) Next(line string) (output iter.Seq[string], err error) {
	if !s.found {
		tsl := strings.TrimSpace(line)
		if tsl == s.lineTS {
			s.found = true
			// keep the current formatting (whitespace) of the line
		} else if _, ok := s.oldLines[tsl]; ok {
			s.found = true
			// replace this with the new version
			return each(s.line), nil
		}
	}
	return each(line), nil
}

// EOF implements Editor.
func (s *simpleEditor) EOF() (output iter.Seq[string], err error) {
	if s.found {
		return empty(), nil
	}
	return each(s.line), nil
}

func empty() iter.Seq[string] {
	return func(func(string) bool) {}
}

func each(v ...string) iter.Seq[string] {
	return func(yield func(string) bool) {
		for _, e := range v {
			if !yield(e) {
				break
			}
		}
	}
}
