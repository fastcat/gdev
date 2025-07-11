package gocache_http

import (
	"net/http"
	"net/url"

	"fastcat.org/go/gdev/addons/gocache"
)

type factory struct{}

// Want implements gocache.RemoteStorageFactory.
func (f factory) Want(uri string) bool {
	u, err := url.Parse(uri)
	if err != nil {
		return false
	}
	return u.Scheme == "http" || u.Scheme == "https"
}

// New implements gocache.RemoteStorageFactory.
func (f factory) New(uri string) (gocache.ReadonlyStorageBackend, error) {
	be, err := newBackend(nil, addon.Config.auth, uri)
	if err != nil {
		return nil, err
	}
	return gocache.DiskDirFromFS(be), nil
}

type nopAuthorizer struct{}

func (nopAuthorizer) Authorize(req *http.Request) error { return nil }
