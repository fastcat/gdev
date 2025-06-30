package gocache

import (
	"errors"
	"fmt"
	"io"
	"sync"
)

type layeredStorageBackend struct {
	local          StorageBackend
	remote         StorageBackend
	remoteReadOnly bool
}

func NewLayeredStorageBackend(local, remote StorageBackend, remoteReadOnly bool) *layeredStorageBackend {
	if local == nil || remote == nil {
		panic(fmt.Errorf(
			"both local and remote storage backends must be non-nil, got local=%T, remote=%T",
			local, remote,
		))
	}
	return &layeredStorageBackend{
		local:          local,
		remote:         remote,
		remoteReadOnly: remoteReadOnly,
	}
}

// CheckOutputFile implements StorageBackend.
func (l *layeredStorageBackend) CheckOutputFile(a ActionEntry) (string, error) {
	panic("unimplemented")
}

// Close implements StorageBackend.
func (l *layeredStorageBackend) Close() error {
	var errs []error
	if l.local != nil {
		if err := l.local.Close(); err != nil {
			errs = append(errs, err)
		} else {
			l.local = nil
		}
	}
	if l.remote != nil {
		if err := l.remote.Close(); err != nil {
			errs = append(errs, err)
		} else {
			l.remote = nil
		}
	}
	return errors.Join(errs...)
}

// ReadActionEntry implements StorageBackend.
func (l *layeredStorageBackend) ReadActionEntry(id []byte) (*ActionEntry, error) {
	// try local first
	a, errLocal := l.local.ReadActionEntry(id)
	if errLocal == nil {
		return a, nil
	}
	// if not found, try remote
	if a, err := l.remote.ReadActionEntry(id); err == nil {
		// if found in remote, write to local
		if err := l.local.WriteActionEntry(*a); err != nil {
			return a, fmt.Errorf("failed to write action entry to local storage: %w", err)
		}
		return a, nil
	}
	// if not found in both, return the local error
	return nil, errLocal
}

// WriteActionEntry implements StorageBackend.
func (l *layeredStorageBackend) WriteActionEntry(a ActionEntry) error {
	var errs []error
	if err := l.local.WriteActionEntry(a); err != nil {
		errs = append(errs, fmt.Errorf("failed to write action entry to local storage: %w", err))
	}
	if !l.remoteReadOnly {
		if err := l.remote.WriteActionEntry(a); err != nil {
			errs = append(errs, fmt.Errorf("failed to write action entry to remote storage: %w", err))
		}
	}
	return errors.Join(errs...)
}

// WriteOutput implements StorageBackend.
func (l *layeredStorageBackend) WriteOutput(a *ActionEntry, body io.Reader) (string, error) {
	if l.remoteReadOnly {
		return l.local.WriteOutput(a, body)
	}

	// write body to both places concurrently
	bodyCopyR, bodyCopyW := io.Pipe()
	defer bodyCopyW.Close() // nolint:errcheck
	defer bodyCopyR.Close() // nolint:errcheck
	body = io.TeeReader(body, bodyCopyW)
	var localPath string
	var err1, err2 error
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		if localPath, err1 = l.local.WriteOutput(a, body); err1 != nil {
			err1 = fmt.Errorf("failed to write output to local storage: %w", err1)
		}
	}()
	go func() {
		defer wg.Done()
		if _, err2 = l.remote.WriteOutput(a, bodyCopyR); err2 != nil {
			err2 = fmt.Errorf("failed to write output to remote storage: %w", err2)
		}
	}()
	wg.Wait()
	return localPath, errors.Join(err1, err2)
}
