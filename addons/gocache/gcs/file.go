package gocache_gcs

import (
	"io/fs"
	"time"

	"cloud.google.com/go/storage"
)

type readerWrapper struct {
	storage.Reader
	name string
}

func (f *readerWrapper) Stat() (fs.FileInfo, error) {
	return &readerFileInfo{f.Attrs, f.name}, nil
}

type readerFileInfo struct {
	storage.ReaderObjectAttrs
	name string
}

func (fi *readerFileInfo) Name() string {
	return fi.name
}

func (fi *readerFileInfo) Size() int64 {
	return fi.ReaderObjectAttrs.Size
}

func (fi *readerFileInfo) Mode() fs.FileMode {
	// not applicable here
	return 0o644
}

func (fi *readerFileInfo) ModTime() time.Time {
	return fi.LastModified
}

func (fi *readerFileInfo) IsDir() bool {
	// not applicable here
	return false
}

func (fi *readerFileInfo) Sys() any {
	// not applicable here
	return nil
}

type writerWrapper struct {
	storage.Writer
}

// Sync implements gocache.WriteFile.
func (w *writerWrapper) Sync() error {
	// no-op for gcs
	return nil
}

type writerFileInfo struct {
	storage.ObjectAttrs
}

func (fi *writerFileInfo) Name() string {
	return fi.ObjectAttrs.Name
}

func (fi *writerFileInfo) Size() int64 {
	return fi.ObjectAttrs.Size
}

func (fi *writerFileInfo) Mode() fs.FileMode {
	// not applicable here
	return 0o644
}

func (fi *writerFileInfo) ModTime() time.Time {
	return fi.Updated
}

func (fi *writerFileInfo) IsDir() bool {
	// not applicable here
	return false
}

func (fi *writerFileInfo) Sys() any {
	// not applicable here
	return nil
}
