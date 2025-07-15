package gocache_gcs

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path"

	"cloud.google.com/go/storage"

	"fastcat.org/go/gdev/addons/gocache"
)

type backend struct {
	ctx        context.Context
	client     *storage.Client
	bucket     *storage.BucketHandle
	bucketName string // BucketHandle has the name, but won't return it
	basePath   string
}

// Close implements gocache.DiskDirFS.
func (g *backend) Close() error {
	if g.client != nil {
		err := g.client.Close()
		g.client = nil
		return err
	}
	return nil
}

// FullName implements gocache.DiskDirFS.
func (g *backend) FullName(name string) string {
	return (&url.URL{
		Scheme: "gs",
		Host:   g.bucketName,
		Path:   path.Join("/"+g.basePath, name),
	}).String()
}

// Mkdir implements gocache.DiskDirFS.
func (g *backend) Mkdir(string, fs.FileMode) error {
	// directories don't (need to) exist in GCS
	return nil
}

// Name implements gocache.DiskDirFS.
func (g *backend) Name() string {
	return (&url.URL{
		Scheme: "gs",
		Host:   g.bucketName,
		Path:   "/" + g.basePath,
	}).String()
}

// Open implements gocache.DiskDirFS.
func (g *backend) Open(name string) (fs.File, error) {
	p := path.Join(g.basePath, name)
	obj := g.bucket.Object(p)
	reader, err := obj.NewReader(g.ctx)
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotExist) {
			err = fmt.Errorf("%w %w", os.ErrNotExist, err)
		}
		return nil, err
	}
	return &readerWrapper{reader, path.Base(p)}, nil
}

// OpenFile implements gocache.DiskDirFS.
func (g *backend) OpenFile(name string, flag int, _ fs.FileMode) (gocache.WriteFile, error) {
	if flag != os.O_WRONLY|os.O_CREATE|os.O_TRUNC {
		return nil, fmt.Errorf("unsupported flag %d", flag)
	}
	p := path.Join(g.basePath, name)
	obj := g.bucket.Object(p)
	return &writerWrapper{*obj.NewWriter(g.ctx)}, nil
}

// Remove implements gocache.DiskDirFS.
func (g *backend) Remove(name string) error {
	p := path.Join(g.basePath, name)
	obj := g.bucket.Object(p)
	if err := obj.Delete(g.ctx); err != nil {
		if errors.Is(err, storage.ErrObjectNotExist) {
			err = fmt.Errorf("%w %w", os.ErrNotExist, err)
		}
		return err
	}
	return nil
}

// Rename implements gocache.DiskDirFS.
func (g *backend) Rename(oldpath, newpath string) error {
	// TODO: the .tmp+rename pattern is costly in GCS
	oldObj := g.bucket.Object(path.Join(g.basePath, oldpath))
	newObj := g.bucket.Object(path.Join(g.basePath, newpath))
	copier := newObj.CopierFrom(oldObj)
	if _, err := copier.Run(g.ctx); err != nil {
		return err
	}
	if err := oldObj.Delete(g.ctx); err != nil && !errors.Is(err, storage.ErrObjectNotExist) {
		return err
	}
	return nil
}

// Stat implements gocache.DiskDirFS.
func (g *backend) Stat(name string) (fs.FileInfo, error) {
	p := path.Join(g.basePath, name)
	obj := g.bucket.Object(p)
	attrs, err := obj.Attrs(g.ctx)
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotExist) {
			err = fmt.Errorf("%w %w", os.ErrNotExist, err)
		}
		return nil, err
	}
	return &fileInfo{*attrs}, nil
}
