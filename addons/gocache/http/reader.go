package gocache_http

import (
	"io/fs"
	"net/http"
	"path"
	"time"
)

type reader struct {
	resp *http.Response
}

// Close implements fs.File.
func (r *reader) Close() error {
	if r.resp != nil {
		// this can't return a meaningful error
		_ = r.resp.Body.Close()
		r.resp = nil
	}
	return nil
}

// Read implements fs.File.
func (r *reader) Read(p []byte) (int, error) {
	return r.resp.Body.Read(p)
}

// Stat implements fs.File.
func (r *reader) Stat() (fs.FileInfo, error) {
	if r.resp == nil {
		return nil, fs.ErrClosed
	}
	return &readerInfo{r.resp}, nil
}

type readerInfo struct {
	resp *http.Response
}

// IsDir implements fs.FileInfo.
//
// It always returns false
func (f *readerInfo) IsDir() bool { return false }

// ModTime implements fs.FileInfo.
func (f *readerInfo) ModTime() time.Time {
	lm := f.resp.Header.Get("Last-Modified")
	if lm == "" {
		return time.Now()
	}
	t, err := time.Parse(time.RFC1123, lm)
	if err != nil {
		// TODO: log?
		return time.Now()
	}
	return t
}

// Mode implements fs.FileInfo.
//
// it always returns 0o400 (read-only)
func (f *readerInfo) Mode() fs.FileMode { return 0o400 }

// Name implements fs.FileInfo.
func (f *readerInfo) Name() string {
	return path.Base(f.resp.Request.URL.Path)
}

// Size implements fs.FileInfo.
func (f *readerInfo) Size() int64 {
	return f.resp.ContentLength
}

// Sys implements fs.FileInfo.
//
// it always returns nil
func (f *readerInfo) Sys() any { return nil }
