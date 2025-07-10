package gocache_gcs

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"cloud.google.com/go/storage"

	"fastcat.org/go/gdev/addons/gocache"
)

type factory struct{}

// Want implements gocache.RemoteStorageFactory.
func (factory) Want(uri string) bool {
	u, err := url.Parse(uri)
	if err != nil {
		return false
	}
	return u.Scheme == "gs"
}

// New implements gocache.RemoteStorageFactory.
func (factory) New(uri string) (gocache.ReadonlyStorageBackend, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}
	if u.User != nil {
		return nil, fmt.Errorf("gcs cache URI %q must not contain user info", uri)
	} else if u.Port() != "" {
		return nil, fmt.Errorf("gcs cache URI %q must not contain port", uri)
	}

	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCS client: %w", err)
	}
	bucket := client.Bucket(u.Host)
	basePath := strings.TrimPrefix(u.Path, "/")

	return gocache.DiskDirFromFS(&backend{ctx, client, bucket, u.Host, basePath}), nil
}
