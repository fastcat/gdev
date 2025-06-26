package gocache

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

type diskDirFS interface {
	fs.FS
	fs.StatFS
	fs.ReadDirFS
}

// DiskDir represents a directory used as part of a cache storage
// implementation.
//
// Even remote caches need a local directory in which to store files.
//
// It uses the same on-disk format as the built-in Go build cache as of Go 1.24.
type DiskDir struct {
	root  diskDirFS
	close func() error
}

func NewDiskDir(path string) (*DiskDir, error) {
	root, err := os.OpenRoot(path)
	if err != nil {
		return nil, err
	}
	return &DiskDir{root: root.FS().(diskDirFS), close: root.Close}, nil
}

func DiskDirFromFS(fs diskDirFS, close func() error) *DiskDir {
	return &DiskDir{root: fs, close: close}
}

func (d *DiskDir) Close() error {
	if d.close != nil {
		if err := d.close(); err != nil {
			return err
		}
	}
	d.close = nil
	d.root = nil
	return nil
}

var (
	ErrDiskStorageClosed   = errors.New("disk storage is closed")
	ErrBadActionFileSize   = errors.New("bad action file size")
	ErrBadActionFileFormat = errors.New("bad action file format")
)

type ActionEntry struct {
	ID       []byte
	OutputID []byte
	Size     int64
	Time     time.Time
}

// GoFileName returns the (relative) path for the given ID and type in the disk
// directory.
func (d *DiskDir) GoFileName(id []byte, typ rune) string {
	return filepath.Join(
		fmt.Sprintf("%02x", id[0]),
		fmt.Sprintf("%x-%c", id, typ),
	)
}

const (
	// action entry file is "v1 <hex id> <hex out> <decimal size space-padded to 20 bytes> <unixnano space-padded to 20 bytes>\n"
	idHashHexSize   = sha256.Size * 2
	actionEntrySize = 2 + 1 + idHashHexSize + 1 + idHashHexSize + 1 + 20 + 1 + 20 + 1
)

// ReadGoActionEntry reads the action data for the given ID from the disk
// directory.
func (d *DiskDir) ReadActionEntry(id []byte) (*ActionEntry, error) {
	f, err := d.root.Open(d.GoFileName(id, 'a'))
	if err != nil {
		return nil, err
	}
	defer f.Close() //nolint:errcheck
	data, err := io.ReadAll(io.LimitReader(f, actionEntrySize))
	if err != nil {
		return nil, err
	}
	parsed, err := parseActionEntry(data)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(id, parsed.ID) {
		return nil, fmt.Errorf("%w: expected ID %x, got %x", ErrBadActionFileFormat, id, parsed.ID)
	}
	return parsed, nil
}

func parseActionEntry(data []byte) (*ActionEntry, error) {
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

var ErrOutputFileWrongSize = errors.New("output file has wrong size")

func (d *DiskDir) CheckOutputFile(a ActionEntry) error {
	st, err := d.root.Stat(d.GoFileName(a.OutputID, 'o'))
	if err != nil {
		return err
	}
	if st.Size() != a.Size {
		return ErrOutputFileWrongSize
	}
	// mtime of output file need not relate to mtime of action file
	return nil
}

// Touch updates the mtime of the action and output file for the given entry, to
// mark them as recently used and prevent them from being trimmed.
func (d *DiskDir) Touch(a ActionEntry) error {
	panic("TODO")
}
