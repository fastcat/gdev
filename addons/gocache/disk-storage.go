package gocache

import (
	"context"
	"errors"
	"os"
	"sync"
	"time"
)

type DiskStorage struct {
	mu      sync.Mutex
	wg      sync.WaitGroup
	closing bool
	dd      *DiskDir
}

type DiskReader DiskStorage

type DiskWriter DiskStorage

var (
	_ Storage      = (*DiskStorage)(nil)
	_ ReadStorage  = (*DiskReader)(nil)
	_ WriteStorage = (*DiskWriter)(nil)
)

func NewDiskStorage(path string) (*DiskStorage, error) {
	dd, err := DiskDirAtRoot(path)
	if err != nil {
		return nil, err
	}
	return &DiskStorage{dd: dd}, nil
}

func NewDiskReader(path string) (*DiskReader, error) {
	if s, err := NewDiskStorage(path); err != nil {
		return nil, err
	} else {
		return (*DiskReader)(s), nil
	}
}

func NewDiskWriter(path string) (*DiskWriter, error) {
	if s, err := NewDiskStorage(path); err != nil {
		return nil, err
	} else {
		return (*DiskWriter)(s), nil
	}
}

// use checks root and closing under the mutex. methods must use it to access
// the root member to ensure it is not closed or removed while they are using
// it.
func (d *DiskStorage) use() (*DiskDir, func(), error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.closing || d.dd == nil || d.dd.root == nil {
		return nil, nil, ErrDiskStorageClosed
	}
	d.wg.Add(1)
	return d.dd, d.wg.Done, nil
}

// Close implements Storage.
func (d *DiskStorage) Close() error {
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
func (d *DiskStorage) Get(ctx context.Context, req *Request) (*Response, error) {
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
func (d *DiskStorage) Put(ctx context.Context, req *Request) (*Response, error) {
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
	outputPath, err := root.WriteOutput(entry, req.Body)
	if err != nil {
		return &Response{
			ID:  req.ID,
			Err: err.Error(),
		}, nil
	}
	if err := root.WriteActionEntry(entry); err != nil {
		return &Response{
			ID:  req.ID,
			Err: err.Error(),
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

// Close implements ReadStorage.
func (d *DiskReader) Close() error {
	return (*DiskStorage)(d).Close()
}

// Get implements ReadStorage.
func (d *DiskReader) Get(ctx context.Context, req *Request) (*Response, error) {
	return (*DiskStorage)(d).Get(ctx, req)
}

// Close implements WriteStorage.
func (d *DiskWriter) Close() error {
	return (*DiskStorage)(d).Close()
}

// Put implements WriteStorage.
func (d *DiskWriter) Put(ctx context.Context, req *Request) (*Response, error) {
	return (*DiskStorage)(d).Put(ctx, req)
}
