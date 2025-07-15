package gocache_s3

import (
	"io/fs"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type readerWrapper struct {
	*s3.GetObjectOutput
	name string
}

// Close implements fs.File.
func (f *readerWrapper) Close() error {
	return f.Body.Close()
}

// Read implements fs.File.
func (f *readerWrapper) Read(p []byte) (int, error) {
	return f.Body.Read(p)
}

// Stat implements fs.File.
func (f *readerWrapper) Stat() (fs.FileInfo, error) {
	return &readerFileInfo{f.GetObjectOutput, f.name}, nil
}

type readerFileInfo struct {
	*s3.GetObjectOutput
	name string
}

func (fi *readerFileInfo) Name() string {
	return fi.name
}

func (fi *readerFileInfo) Size() int64 {
	if fi.ContentLength != nil {
		return *fi.ContentLength
	}
	return 0
}

func (fi *readerFileInfo) Mode() fs.FileMode {
	// not applicable here
	return 0o644
}

func (fi *readerFileInfo) ModTime() time.Time {
	if fi.LastModified != nil {
		return *fi.LastModified
	}
	return time.Time{}
}

func (fi *readerFileInfo) IsDir() bool {
	// not applicable here
	return false
}

func (fi *readerFileInfo) Sys() any {
	// not applicable here
	return nil
}

type fileInfo struct {
	*s3.HeadObjectOutput
	name string
}

// IsDir implements fs.FileInfo.
func (f *fileInfo) IsDir() bool {
	// not applicable here
	return false
}

// ModTime implements fs.FileInfo.
func (f *fileInfo) ModTime() time.Time {
	if f.LastModified != nil {
		return *f.LastModified
	}
	return time.Time{}
}

// Mode implements fs.FileInfo.
func (f *fileInfo) Mode() fs.FileMode {
	// not applicable here
	return 0o644
}

// Name implements fs.FileInfo.
func (f *fileInfo) Name() string {
	return f.name
}

// Size implements fs.FileInfo.
func (f *fileInfo) Size() int64 {
	if f.ContentLength != nil {
		return *f.ContentLength
	}
	return 0
}

// Sys implements fs.FileInfo.
func (f *fileInfo) Sys() any {
	// not applicable here
	return nil
}
