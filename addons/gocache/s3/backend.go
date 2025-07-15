package gocache_s3

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"os"
	"path"
	"sync/atomic"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"

	"fastcat.org/go/gdev/addons/gocache"
)

type backend struct {
	ctx        context.Context
	client     *s3.Client
	bucketName string
	basePath   string
	cantRename atomic.Bool
}

// Close implements gocache.DiskDirFS.
func (b *backend) Close() error {
	if b.client != nil {
		// no close for S3 client
		b.client = nil
	}
	return nil
}

// FullName implements gocache.DiskDirFS.
func (b *backend) FullName(name string) string {
	return (&url.URL{
		Scheme: "s3",
		Host:   b.bucketName,
		Path:   path.Join("/"+b.basePath, name),
	}).String()
}

// Mkdir implements gocache.DiskDirFS.
func (b *backend) Mkdir(path string, mode fs.FileMode) error {
	// directories don't (need to) exist in S3
	return nil
}

// Name implements gocache.DiskDirFS.
func (b *backend) Name() string {
	return (&url.URL{
		Scheme: "s3",
		Host:   b.bucketName,
		Path:   "/" + b.basePath,
	}).String()
}

// Open implements gocache.DiskDirFS.
func (b *backend) Open(name string) (fs.File, error) {
	p := path.Join(b.basePath, name)
	resp, err := b.client.GetObject(b.ctx, &s3.GetObjectInput{
		Bucket: &b.bucketName,
		Key:    &p,
	})
	if err != nil {
		err = translateErr(err)
		return nil, err
	}
	return &readerWrapper{resp, path.Base(p)}, nil
}

// OpenFile implements gocache.DiskDirFS.
func (b *backend) OpenFile(name string, flag int, perm fs.FileMode) (gocache.WriteFile, error) {
	if flag != os.O_WRONLY|os.O_CREATE|os.O_TRUNC {
		return nil, fmt.Errorf("unsupported flag %d", flag)
	}
	p := path.Join(b.basePath, name)
	pr, pw := io.Pipe()
	w := &writer{
		ctx:        b.ctx,
		c:          b.client,
		bucketName: b.bucketName,
		key:        p,
		pr:         pr,
		pw:         pw,
	}
	w.start()
	return w, nil
}

// Remove implements gocache.DiskDirFS.
func (b *backend) Remove(name string) error {
	p := path.Join(b.basePath, name)
	if _, err := b.client.DeleteObject(b.ctx, &s3.DeleteObjectInput{
		Bucket: &b.bucketName,
		Key:    &p,
	}); err != nil {
		return translateErr(err)
	}
	return nil
}

// Rename implements gocache.DiskDirFS.
func (b *backend) Rename(oldpath, newpath string) error {
	// TODO: the .tmp+rename pattern is costly in S3
	oldObj := path.Join(b.basePath, oldpath)
	newObj := path.Join(b.basePath, newpath)

	if !b.cantRename.Load() {
		_, err := b.client.RenameObject(b.ctx, &s3.RenameObjectInput{
			Bucket:       &b.bucketName,
			Key:          &newpath,
			RenameSource: &oldObj,
		})
		if err != nil {
			var opErr smithy.APIError
			if errors.As(err, &opErr) && opErr.ErrorCode() == "NotImplemented" {
				// this bucket doesn't support renames
				b.cantRename.Store(true)
				// fall through to the copy+delete strategy
			} else {
				return translateErr(err)
			}
		}
	}
	// copy + delete
	if _, err := b.client.CopyObject(b.ctx, &s3.CopyObjectInput{
		Bucket: &b.bucketName,
		// old object needs to have the (source) bucket as part of it
		CopySource: aws.String(path.Join(b.bucketName, oldObj)),
		Key:        &newObj,
	}); err != nil {
		return translateErr(err)
	}
	if _, err := b.client.DeleteObject(b.ctx, &s3.DeleteObjectInput{
		Bucket: &b.bucketName,
		Key:    &oldObj,
	}); err != nil {
		return translateErr(err)
	}
	return nil
}

// Stat implements gocache.DiskDirFS.
func (b *backend) Stat(name string) (fs.FileInfo, error) {
	p := path.Join(b.basePath, name)
	resp, err := b.client.HeadObject(b.ctx, &s3.HeadObjectInput{
		Bucket: &b.bucketName,
		Key:    &p,
	})
	if err != nil {
		return nil, translateErr(err)
	}
	return &fileInfo{resp, path.Base(p)}, nil
}

func translateErr(err error) error {
	var nfe *types.NotFound
	var nsk *types.NoSuchKey
	if errors.As(err, &nfe) {
		return fmt.Errorf("%w %w", os.ErrNotExist, err)
	} else if errors.As(err, &nsk) {
		return fmt.Errorf("%w %w", os.ErrNotExist, err)
	}
	return err
}
