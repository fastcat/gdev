package gocache_s3

import (
	"bytes"
	"context"
	"errors"
	"io"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// similar to gocache_http.writer

// use the minimum valid part size.
const chunkSize = 5 * 1024 * 1024

var chunkPool = &sync.Pool{
	New: func() any {
		return bytes.NewBuffer(make([]byte, 0, chunkSize))
	},
}

type writer struct {
	ctx             context.Context
	c               *s3.Client
	bucketName, key string
	uploadID        string
	run             chan struct{}
	errCh           chan error
	pr              *io.PipeReader
	pw              *io.PipeWriter
	parts           []types.CompletedPart
}

func (w *writer) start() {
	w.run = make(chan struct{})
	w.errCh = make(chan error, 3)
	go w.do()
	// wait for it to be ready, it won't be safe to call Close() until this
	<-w.run
}

func (w *writer) do() {
	run, errCh := w.run, w.errCh
	defer close(errCh)
	// tell start we have read all the race-prone members we need
	run <- struct{}{}

	chunkBuf := chunkPool.Get().(*bytes.Buffer)
	defer chunkPool.Put(chunkBuf)

	// upload chunks
	for partNum := int32(1); ; partNum++ {
		chunkBuf.Reset()
		// CopyNBuffer
		n, crErr := io.CopyN(chunkBuf, io.LimitReader(w.pr, chunkSize), chunkSize)
		if n > 0 {
			// upload the chunk
			resp, err := w.c.UploadPart(w.ctx, &s3.UploadPartInput{
				Bucket:     &w.bucketName,
				Key:        &w.key,
				PartNumber: &partNum,
				UploadId:   &w.uploadID,
				Body:       bytes.NewReader(chunkBuf.Bytes()[:n]),
			})
			if err != nil {
				errCh <- err
				return
			}
			w.parts = append(w.parts, types.CompletedPart{
				PartNumber: aws.Int32(partNum),
				// ETag is required
				ETag: resp.ETag,
				// Including checksums seems to break things
			})
		}
		if crErr != nil {
			if !errors.Is(crErr, io.EOF) {
				errCh <- crErr
			}
			return
		}
	}
}

// Close implements gocache.WriteFile.
func (w *writer) Close() error {
	var errs []error
	// we have to be careful about races for the first part of this
	if w.run != nil {
		close(w.run)
		w.run = nil
	}
	if w.pw != nil {
		errs = append(errs, w.pw.Close())
		w.pw = nil
	}
	// wait for the background goroutine to finish
	if w.errCh != nil {
		for err := range w.errCh {
			errs = append(errs, translateNotFound(err))
		}
	}
	if w.uploadID != "" {
		// TODO: abort the upload if there was an error writing
		// complete the upload
		if _, err := w.c.CompleteMultipartUpload(w.ctx, &s3.CompleteMultipartUploadInput{
			Bucket:   &w.bucketName,
			Key:      &w.key,
			UploadId: &w.uploadID,
			MultipartUpload: &types.CompletedMultipartUpload{
				Parts: w.parts,
			},
		}); err != nil {
			errs = append(errs, translateNotFound(err))
		}
		w.uploadID = ""
	}
	// leave errCh alone, it'll be closed by now
	return errors.Join(errs...)
}

// Sync implements gocache.WriteFile.
//
// It closes the request body and waits to get the response from the server to
// catch permission type errors and the like.
func (w *writer) Sync() error {
	if w.pw != nil {
		if err := w.pw.Close(); err != nil {
			return err
		}
		w.pw = nil
	}
	var errs []error
	if w.errCh != nil {
		for err := range w.errCh {
			if err != nil {
				errs = append(errs, translateNotFound(err))
			}
		}
	}
	return errors.Join(errs...)
}

// Write implements gocache.WriteFile.
func (w *writer) Write(data []byte) (n int, err error) {
	return w.pw.Write(data)
}
