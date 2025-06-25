package gocache

import (
	"context"
	"fmt"
	"os"
	"sync"
)

type DiskStorage struct {
	mu      sync.Mutex
	wg      sync.WaitGroup
	closing bool
	root    *os.Root
}

type DiskReader DiskStorage

type DiskWriter DiskStorage

var (
	_ Storage      = (*DiskStorage)(nil)
	_ ReadStorage  = (*DiskReader)(nil)
	_ WriteStorage = (*DiskWriter)(nil)
)

func NewDiskStorage(path string) (*DiskStorage, error) {
	root, err := os.OpenRoot(path)
	if err != nil {
		return nil, err
	}
	return &DiskStorage{root: root}, nil
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

	if d.root != nil {
		if err := d.root.Close(); err != nil {
			return err
		}
		d.root = nil
	}

	return nil
}

var ErrDiskStorageClosed = fmt.Errorf("disk storage is closed")

// use checks root and closing under the mutex. methods must use it to access
// the root member to ensure it is not closed or removed while they are using
// it.
func (d *DiskStorage) use() (*os.Root, func(), error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.root == nil || d.closing {
		return nil, nil, ErrDiskStorageClosed
	}
	d.wg.Add(1)
	return d.root, d.wg.Done, nil
}

// Get implements Storage.
func (d *DiskStorage) Get(ctx context.Context, req *Request) (*Response, error) {
	root, done, err := d.use()
	if err != nil {
		return nil, err
	}
	defer done()

	_ = root

	panic("unimplemented")
}

// Put implements Storage.
func (d *DiskStorage) Put(ctx context.Context, req *Request) (*Response, error) {
	root, done, err := d.use()
	if err != nil {
		return nil, err
	}
	defer done()

	_ = root

	panic("unimplemented")
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
