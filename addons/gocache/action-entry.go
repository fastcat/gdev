package gocache

import (
	"fmt"
	"io"
	"time"
)

type ActionEntry struct {
	ID       []byte
	OutputID []byte
	Size     int64
	Time     time.Time
}

func ParseActionEntry(data []byte) (*ActionEntry, error) {
	if len(data) != actionEntrySize {
		return nil, fmt.Errorf("%w: expect %d, got at least %d", ErrBadActionFileSize, actionEntrySize, len(data))
	}
	var parsed ActionEntry
	var timeNanos int64
	if n, err := fmt.Sscanf(
		string(data),
		"v1 %x %x %d %d\n",
		&parsed.ID, &parsed.OutputID, &parsed.Size, &timeNanos,
	); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrBadActionFileFormat, err)
	} else if n != 4 {
		return nil, fmt.Errorf("%w: expected 4 fields, got %d", ErrBadActionFileFormat, n)
	}
	parsed.Time = time.Unix(0, timeNanos)
	return &parsed, nil
}

func (a ActionEntry) WriteTo(w io.Writer) (int64, error) {
	n, err := fmt.Fprintf(w,
		"v1 %x %x %20d %20d\n",
		a.ID, a.OutputID, a.Size, a.Time.UnixNano(),
	)
	if err != nil {
		return int64(n), err
	} else if n != actionEntrySize {
		return int64(n), fmt.Errorf("%w: expected %d bytes, wrote %d", ErrBadActionFileSize, actionEntrySize, n)
	}
	return int64(n), nil
}
