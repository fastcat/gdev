package gocache

import (
	"context"
	"errors"
	"os"
	"sync"
	"time"
)

type StorageFrontend struct {
	mu      sync.Mutex
	wg      sync.WaitGroup
	closing bool
	dd      StorageBackend
}

type ReadonlyStorageFrontend StorageFrontend

var (
	_ Storage     = (*StorageFrontend)(nil)
	_ ReadStorage = (*ReadonlyStorageFrontend)(nil)
)

func NewDiskStorage(path string) (*StorageFrontend, error) {
	dd, err := DiskDirAtRoot(path)
	if err != nil {
		return nil, err
	}
	return &StorageFrontend{dd: dd}, nil
}

func NewDiskReader(path string) (*ReadonlyStorageFrontend, error) {
	if s, err := NewDiskStorage(path); err != nil {
		return nil, err
	} else {
		return (*ReadonlyStorageFrontend)(s), nil
	}
}

func NewFrontend(backend StorageBackend) *StorageFrontend {
	if backend == nil {
		panic("backend must not be nil")
	}
	return &StorageFrontend{dd: backend}
}

// TODO: accept a read-only backend
func NewReadonlyFrontend(backend StorageBackend) *ReadonlyStorageFrontend {
	if backend == nil {
		panic("backend must not be nil")
	}
	return (*ReadonlyStorageFrontend)(&StorageFrontend{dd: backend})
}

// use checks root and closing under the mutex. methods must use it to access
// the root member to ensure it is not closed or removed while they are using
// it.
func (d *StorageFrontend) use() (StorageBackend, func(), error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.closing || d.dd == nil {
		return nil, nil, ErrDiskStorageClosed
	}
	d.wg.Add(1)
	return d.dd, d.wg.Done, nil
}

// Close implements Storage.
func (d *StorageFrontend) Close() error {
	// stop new stuff from starting
	d.mu.Lock()
	d.closing = true
	d.mu.Unlock()

	// wait for existing stuff to finish without lock held
	d.wg.Wait()

	d.mu.Lock()
	defer d.mu.Unlock()

	if d.dd != nil {
		if err := d.dd.Close(); err != nil {
			return err
		}
	}

	return nil
}

// Get implements Storage.
func (d *StorageFrontend) Get(ctx context.Context, req *Request) (*Response, error) {
	root, done, err := d.use()
	if err != nil {
		return nil, err
	}
	defer done()

	entry, err := root.ReadActionEntry(req.ActionID)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// cache miss
			return &Response{
				ID:   req.ID,
				Miss: true,
			}, nil
		}
		// other error
		return &Response{
			ID:   req.ID,
			Err:  err.Error(),
			Miss: true,
		}, nil
	}
	outputPath, err := root.CheckOutputFile(*entry)
	if err != nil {
		return &Response{
			ID:   req.ID,
			Err:  err.Error(),
			Miss: true,
		}, nil
	} else if outputPath == "" {
		// remote read-only storage, we can't use the output file we have :(
		return &Response{
			ID:   req.ID,
			Miss: true,
		}, nil
	}
	return &Response{
		ID:       req.ID,
		OutputID: entry.OutputID,
		Size:     entry.Size,
		Time:     &entry.Time,
		DiskPath: outputPath,
	}, nil
}

// Put implements Storage.
func (d *StorageFrontend) Put(ctx context.Context, req *Request) (*Response, error) {
	root, done, err := d.use()
	if err != nil {
		return nil, err
	}
	defer done()

	entry := ActionEntry{
		ID:       req.ActionID,
		OutputID: req.OutputID,
		Size:     req.BodySize,
		Time:     time.Now(),
	}
	res := Response{
		ID:       req.ID,
		OutputID: entry.OutputID,
		Size:     entry.Size,
		Time:     &entry.Time,
	}
	if req.Body != nil {
		res.DiskPath, err = root.WriteOutput(entry, req.Body)
		if err != nil {
			return &Response{
				ID:  req.ID,
				Err: err.Error(),
			}, nil
		}
	}
	if err := root.WriteActionEntry(entry); err != nil {
		return &Response{
			ID:  req.ID,
			Err: err.Error(),
		}, nil
	}

	return &res, nil
}

// Close implements ReadStorage.
func (d *ReadonlyStorageFrontend) Close() error {
	return (*StorageFrontend)(d).Close()
}

// Get implements ReadStorage.
func (d *ReadonlyStorageFrontend) Get(ctx context.Context, req *Request) (*Response, error) {
	return (*StorageFrontend)(d).Get(ctx, req)
}
