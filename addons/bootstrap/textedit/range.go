package textedit

import (
	"fmt"
	"iter"
	"strings"
)

// Splice a range of lines marked by the start and end lines. Panics if
// len(lines) < 3, i.e. must have start & end markers and at least one "content"
// line, or if the start or end markers are empty after trimming leading &
// trailing whitespace.
//
// Matching the start and end markers is done after trimming leading & trailing
// whitespace from both the sought & observed lines.
func SpliceRange(lines ...string) Editor {
	if len(lines) < 3 {
		panic(fmt.Errorf("SpliceRange requires at least 3 lines: start, content, end"))
	}
	start, end := lines[0], lines[len(lines)-1]
	startTS, endTS := strings.TrimSpace(start), strings.TrimSpace(end)
	if startTS == "" || endTS == "" {
		panic(fmt.Errorf("SpliceRange start and end markers must not be empty after trimming whitespace"))
	}
	content := lines[1 : len(lines)-1]
	return &rangeEditor{
		start:   start,
		end:     end,
		startTS: startTS,
		endTS:   endTS,
		content: content,
	}
}

type rangeEditor struct {
	start, end     string
	startTS, endTS string
	content        []string

	startFound, endFound bool
}

// Next implements Editor.
func (r *rangeEditor) Next(line string) (output iter.Seq[string], err error) {
	tsl := strings.TrimSpace(line)
	if r.endFound {
		// already did the edit
		return each(line), nil
	} else if r.startFound {
		if tsl == r.endTS {
			r.endFound = true
			rr := make([]string, 0, 2+len(r.content))
			rr = append(rr, r.start)
			rr = append(rr, r.content...)
			rr = append(rr, r.end)
			return each(rr...), nil
		} else {
			// emit nothing while in between start & end markers
			return empty(), nil
		}
	} else if tsl == r.startTS {
		r.startFound = true
		// emit nothing until we find the end marker
		return empty(), nil
	} else {
		// not in the range, just copy the line
		return each(line), nil
	}
}

// EOF implements Editor.
func (r *rangeEditor) EOF() (output iter.Seq[string], err error) {
	if !r.endFound {
		// we didn't find the end marker, whether or not we found the start marker
		// we need to emit the range
		rr := make([]string, 0, 2+len(r.content))
		rr = append(rr, r.start)
		rr = append(rr, r.content...)
		rr = append(rr, r.end)
		return each(rr...), nil
	}
	return empty(), nil
}
