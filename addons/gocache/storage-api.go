package gocache

import (
	"context"
	"io"
)

// ReadStorage defines the interface for the backend storage engine servicing
// [Request] instances received that supports [CmdGet] and [CmdPut].
type ReadStorage interface {
	io.Closer
	Get(
		ctx context.Context,
		req *Request,
	) (*Response, error)
}

// WriteStorage defines the interface for the backend storage engine servicing
// [Request] instances received that supports [CmdPut] and [CmdClose].
type WriteStorage interface {
	io.Closer
	Put(
		ctx context.Context,
		req *Request,
	) (*Response, error)
}

// Storage is the interface that combines [ReadStorage] and [WriteStorage] to
// service [Request] instances received that supports [CmdGet], [CmdPut], and
// [CmdClose].
type Storage interface {
	ReadStorage
	WriteStorage
}

type StorageBackend interface {
	io.Closer
	ReadActionEntry(id []byte) (*ActionEntry, error)
	CheckOutputFile(a ActionEntry) (string, error)
	// may update a.Time
	WriteOutput(a *ActionEntry, body io.Reader) (string, error)
	WriteActionEntry(a ActionEntry) error
}
