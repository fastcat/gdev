package gocache_s3

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"fastcat.org/go/gdev/addons/gocache"

	aws_config "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type factory struct{}

// Name implements gocache.RemoteStorageFactory.
func (factory) Name() string {
	return "s3"
}

// Want implements gocache.RemoteStorageFactory.
func (factory) Want(uri string) bool {
	u, err := url.Parse(uri)
	if err != nil {
		return false
	}
	// TODO: S3 scheme requires region specification elsewhere
	return u.Scheme == "s3"
	// TODO: recognize http://BUCKET.s3-REGION.amazonaws.com
	// TODO: recognize https://s3-REGION.amazonaws.com/BUCKET
}

// New implements gocache.RemoteStorageFactory.
func (factory) New(uri string) (gocache.ReadonlyStorageBackend, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}
	if u.User != nil {
		return nil, fmt.Errorf("s3 cache URI %q must not contain user info", uri)
	} else if u.Port() != "" {
		return nil, fmt.Errorf("s3 cache URI %q must not contain port", uri)
	}

	ctx := context.Background()
	cfg, err := aws_config.LoadDefaultConfig(ctx,
		aws_config.WithRegion(addon.Config.region),
	)
	if err != nil {
		return nil, err
	}
	client := s3.NewFromConfig(cfg)
	basePath := strings.TrimPrefix(u.Path, "/")

	return gocache.DiskDirFromFS(&backend{
		ctx:        ctx,
		client:     client,
		bucketName: u.Host,
		basePath:   basePath,
	}), nil
}
