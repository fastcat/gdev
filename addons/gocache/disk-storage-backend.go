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
)

type diskDirBaseFS interface {
	fs.FS
	fs.StatFS
}

type writeFile interface {
	io.WriteCloser
	Sync() error
}

type diskDirFS interface {
	io.Closer
	diskDirBaseFS
	Name() string
	FullName(string) string
	OpenFile(name string, flag int, perm fs.FileMode) (writeFile, error)
	Rename(oldpath, newpath string) error
	Remove(name string) error
	Mkdir(path string, mode fs.FileMode) error
}

// diskStorageBackend represents a directory used as part of a cache storage
// implementation.
//
// Even remote caches need a local directory in which to store files.
//
// It uses the same on-disk format as the built-in Go build cache as of Go 1.24.
type diskStorageBackend struct {
	root diskDirFS
}

func DiskDirAtRoot(path string) (*diskStorageBackend, error) {
	root, err := os.OpenRoot(path)
	if err != nil {
		return nil, err
	}
	return &diskStorageBackend{root: wrapRoot(root)}, nil
}

func DiskDirFromFS(fs diskDirFS, close func() error) *diskStorageBackend {
	return &diskStorageBackend{root: fs}
}

func (d *diskStorageBackend) Close() error {
	if d.root != nil {
		if err := d.root.Close(); err != nil {
			return err
		}
		d.root = nil
	}
	return nil
}

var (
	ErrDiskStorageClosed   = errors.New("disk storage is closed")
	ErrBadActionFileSize   = errors.New("bad action file size")
	ErrBadActionFileFormat = errors.New("bad action file format")
)

// GoFileName returns the (relative) path for the given ID and type in the disk
// directory.
func (d *diskStorageBackend) GoFileName(id []byte, typ rune) string {
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
func (d *diskStorageBackend) ReadActionEntry(id []byte) (*ActionEntry, error) {
	f, err := d.root.Open(d.GoFileName(id, 'a'))
	if err != nil {
		return nil, err
	}
	defer f.Close() //nolint:errcheck
	data, err := io.ReadAll(io.LimitReader(f, actionEntrySize))
	if err != nil {
		return nil, err
	}
	parsed, err := ParseActionEntry(data)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(id, parsed.ID) {
		return nil, fmt.Errorf("%w: expected ID %x, got %x", ErrBadActionFileFormat, id, parsed.ID)
	}
	return parsed, nil
}

var ErrOutputFileWrongSize = errors.New("output file has wrong size")

func (d *diskStorageBackend) CheckOutputFile(a ActionEntry) (string, error) {
	fn := d.GoFileName(a.OutputID, 'o')
	st, err := d.root.Stat(fn)
	if err != nil {
		return d.root.FullName(fn), err
	}
	if st.Size() != a.Size {
		return d.root.FullName(fn), ErrOutputFileWrongSize
	}
	// mtime of output file need not relate to mtime of action file
	return d.root.FullName(fn), nil
}

func (d *diskStorageBackend) WriteOutput(a ActionEntry, body io.Reader) (string, error) {
	fn := d.GoFileName(a.OutputID, 'o')
	// if it looks like the right size, and isn't newer than the action entry, we
	// can skip writing the file
	if st, err := d.root.Stat(fn); err == nil {
		if st.Size() == a.Size && !st.ModTime().After(a.Time) {
			// drain body that we aren't using, to ensure caller doesn't get messed up
			// state
			_, err = io.Copy(io.Discard, body) // nolint:errcheck
			return d.root.FullName(fn), err
		}
	}

	if err := d.root.Mkdir(filepath.Dir(fn), 0o755); err != nil && !errors.Is(err, fs.ErrExist) {
		return d.root.FullName(fn), fmt.Errorf("failed to create directory %q: %w", filepath.Dir(fn), err)
	}

	f, err := d.root.OpenFile(fn+".tmp", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return d.root.FullName(fn), err
	}
	defer f.Close() //nolint:errcheck

	if n, err := io.Copy(f, body); err != nil {
		return d.root.FullName(fn), err
	} else if n != a.Size {
		return d.root.FullName(fn), fmt.Errorf("%w: expected %d bytes, got %d", ErrOutputFileWrongSize, a.Size, n)
	}
	if err := f.Sync(); err != nil {
		return d.root.FullName(fn), err
	}
	if err := f.Close(); err != nil {
		return d.root.FullName(fn), err
	}
	// TODO: go 1.24 os.Root.Rename
	if err := d.root.Rename(fn+".tmp", fn); err != nil {
		if err2 := os.Remove(fn + ".tmp"); err2 != nil {
			err = errors.Join(err, err2)
		}
		return d.root.FullName(fn), err
	}

	return d.root.FullName(fn), nil
}

func (d *diskStorageBackend) OpenOutputFile(a ActionEntry) (io.ReadCloser, error) {
	fn := d.GoFileName(a.OutputID, 'o')
	return d.root.Open(fn)
}

func (d *diskStorageBackend) WriteActionEntry(a ActionEntry) error {
	fn := d.GoFileName(a.ID, 'a')
	if err := d.root.Mkdir(filepath.Dir(fn), 0o755); err != nil && !errors.Is(err, fs.ErrExist) {
		return fmt.Errorf("failed to create directory %q: %w", filepath.Dir(fn), err)
	}
	f, err := d.root.OpenFile(fn+".tmp", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer f.Close() //nolint:errcheck
	if _, err := a.WriteTo(f); err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := d.root.Rename(fn+".tmp", fn); err != nil {
		if err2 := d.root.Remove(fn + ".tmp"); err2 != nil {
			err = errors.Join(err, err2)
		}
		return err
	}
	return nil
}
