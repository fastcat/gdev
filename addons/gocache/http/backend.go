package gocache_http

import (
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"

	"fastcat.org/go/gdev/addons/gocache"
)

type backend struct {
	c    *http.Client
	a    Authorizer
	base *url.URL
}

func newBackend(
	c *http.Client,
	a Authorizer,
	base string,
) (*backend, error) {
	u, err := url.Parse(base)
	if err != nil {
		return nil, err
	} else if !u.IsAbs() {
		return nil, fmt.Errorf("base URL must be absolute: %q", base)
	} else if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("invalid scheme for http cache: %q", u.Scheme)
	}
	if !strings.HasSuffix(u.Path, "/") {
		u.Path += "/"
	}
	if c == nil {
		c = http.DefaultClient
	}
	if a == nil {
		a = nopAuthorizer{}
	}
	return &backend{
		c:    c,
		a:    a,
		base: u,
	}, nil
}

// Close implements gocache.DiskDirFS.
func (b *backend) Close() error {
	return nil
}

// FullName implements gocache.DiskDirFS.
func (b *backend) FullName(name string) string {
	return b.fullURL(name).String()
}

func (b *backend) fullURL(name string) *url.URL {
	return b.base.ResolveReference(&url.URL{Path: path.Clean("./" + name)})
}

// Mkdir implements gocache.DiskDirFS.
func (b *backend) Mkdir(string, fs.FileMode) error {
	// inapplicable for HTTP
	return nil
}

// Name implements gocache.DiskDirFS.
func (b *backend) Name() string {
	return b.base.String()
}

// Open implements gocache.DiskDirFS.
func (b *backend) Open(name string) (fs.File, error) {
	u := b.fullURL(name)
	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request for %q: %w", u, err)
	}
	if err := b.a.Authorize(req); err != nil {
		return nil, fmt.Errorf("failed to authorize request for %q: %w", u, err)
	}
	resp, err := b.c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to open %q: %w", u, err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		if err := statusError(resp.StatusCode); err != nil {
			return nil, fmt.Errorf("failed to open %q: %s: %w", u, resp.Status, err)
		}
		return nil, fmt.Errorf("failed to open %q: %s", u, resp.Status)
	}
	return &reader{resp: resp}, nil
}

func statusError(code int) error {
	switch code {
	case http.StatusNotFound:
		return os.ErrNotExist
	case http.StatusUnauthorized, http.StatusForbidden:
		return os.ErrPermission
	default:
		return nil
	}
}

// OpenFile implements gocache.DiskDirFS.
//
// It only supports write-only mode.
//
// Due to the limitations of HTTP, it can't report permission and other errors
// until _after_ the file is written and closed.
func (b *backend) OpenFile(name string, flag int, _ fs.FileMode) (gocache.WriteFile, error) {
	if flag != os.O_WRONLY|os.O_CREATE|os.O_TRUNC {
		return nil, fmt.Errorf("unsupported flag %d", flag)
	}
	u := b.fullURL(name)
	pr, pw := io.Pipe()
	req, err := http.NewRequest(http.MethodPut, u.String(), pr)
	if err != nil {
		return nil, fmt.Errorf("failed to create request for %q: %w", u, err)
	}
	w := &writer{c: b.c, req: req, pw: pw}
	w.start()
	return w, nil
}

// Remove implements gocache.DiskDirFS.
func (b *backend) Remove(name string) error {
	u := b.fullURL(name)
	req, err := http.NewRequest(http.MethodDelete, u.String(), nil)
	if err != nil {
		return fmt.Errorf("failed to create request for %q: %w", u, err)
	}
	if err := b.a.Authorize(req); err != nil {
		return fmt.Errorf("failed to authorize request for %q: %w", u, err)
	}
	resp, err := b.c.Do(req)
	if err != nil {
		return fmt.Errorf("failed to remove %q: %w", u, err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		resp.Body.Close()
		return fmt.Errorf("failed to remove %q: %s", u, resp.Status)
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	return nil
}

// Rename implements gocache.DiskDirFS.
//
// It requires the server to support the WebDAV MOVE method using an absolute
// path (but not an absolute URL) for the Destination.
func (b *backend) Rename(oldpath string, newpath string) error {
	u := b.fullURL(oldpath)
	newURL := b.fullURL(newpath)
	req, err := http.NewRequest("MOVE", u.String(), nil)
	if err != nil {
		return fmt.Errorf("failed to create request for %q: %w", u, err)
	}
	req.Header.Set("Destination", newURL.Path)
	req.Header.Set("Overwrite", "T")
	if err := b.a.Authorize(req); err != nil {
		return fmt.Errorf("failed to authorize request for %q: %w", u, err)
	}
	resp, err := b.c.Do(req)
	if err != nil {
		return fmt.Errorf("failed to rename %q to %q: %w", u, newURL, err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		resp.Body.Close()
		return fmt.Errorf("failed to rename %q to %q: %s", u, newURL, resp.Status)
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	return nil
}

// Stat implements gocache.DiskDirFS.
func (b *backend) Stat(name string) (fs.FileInfo, error) {
	u := b.fullURL(name)
	req, err := http.NewRequest(http.MethodHead, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request for %q: %w", u, err)
	}
	resp, err := b.c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to stat %q: %w", u, err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		resp.Body.Close()
		return nil, fmt.Errorf("failed to stat %q: %s", u, resp.Status)
	}
	return &readerInfo{resp: resp}, nil
}
