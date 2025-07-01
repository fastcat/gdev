package gocache

import (
	"errors"
	"fmt"
	"io"
	"sync"
)

type layeredStorageBackend struct {
	local   ReadonlyStorageBackend
	localW  StorageBackend
	remote  ReadonlyStorageBackend
	remoteW StorageBackend
}

func NewReadonlyStorageBackend(local, remote ReadonlyStorageBackend) *readonlyLayeredStorageBackend {
	if local == nil || remote == nil {
		panic(fmt.Errorf(
			"both local and remote storage backends must be non-nil, got local=%T, remote=%T",
			local, remote,
		))
	}
	ret := &readonlyLayeredStorageBackend{
		local:  local,
		remote: remote,
	}
	return ret
}

func NewReadThroughStorageBackend(
	local StorageBackend,
	remote ReadonlyStorageBackend,
) *layeredStorageBackend {
	if local == nil || remote == nil {
		panic(fmt.Errorf(
			"both local and remote storage backends must be non-nil, got local=%T, remote=%T",
			local, remote,
		))
	}
	ret := &layeredStorageBackend{
		local:  local,
		localW: local,
		remote: remote,
	}
	return ret
}

func NewWriteThroughStorageBackend(local, remote StorageBackend) *layeredStorageBackend {
	if local == nil || remote == nil {
		panic(fmt.Errorf(
			"both local and remote storage backends must be non-nil, got local=%T, remote=%T",
			local, remote,
		))
	}
	ret := &layeredStorageBackend{
		local:   local,
		localW:  local,
		remote:  remote,
		remoteW: remote,
	}
	return ret
}

// CheckOutputFile implements StorageBackend.
func (l *layeredStorageBackend) CheckOutputFile(a ActionEntry) (string, error) {
	fullFn, localErr := l.local.CheckOutputFile(a)
	if localErr == nil {
		// already present locally
		return fullFn, nil
	}
	if _, err := l.remote.CheckOutputFile(a); err == nil {
		if l.localW == nil {
			// the output file exists but we can't use it because there's nowhere
			// local to write it to
			return "", nil
		}
		f, err := l.remote.OpenOutputFile(a)
		if err == nil {
			defer f.Close() // nolint:errcheck
			if fullFn, err = l.localW.WriteOutput(a, f); err == nil {
				// if we successfully wrote to local, return the local path
				// TODO: a.time becomes wrong
				return fullFn, nil
			}
		}
	}
	// if not found in remote, return the local error
	return fullFn, localErr
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
		if l.localW != nil {
			// if found in remote, write to local
			if err := l.localW.WriteActionEntry(*a); err != nil {
				return a, fmt.Errorf("failed to write action entry to local storage: %w", err)
			}
		}
		return a, nil
	}
	// if not found in both, return the local error
	return nil, errLocal
}

var ErrReadonlyStorage = fmt.Errorf("readonly storage backend")

// WriteActionEntry implements StorageBackend.
func (l *layeredStorageBackend) WriteActionEntry(a ActionEntry) error {
	if l.localW == nil {
		// should not be reachable
		panic(ErrReadonlyStorage)
	}
	var errs []error
	if err := l.localW.WriteActionEntry(a); err != nil {
		errs = append(errs, fmt.Errorf("failed to write action entry to local storage: %w", err))
	}
	if l.remoteW != nil {
		if err := l.remoteW.WriteActionEntry(a); err != nil {
			errs = append(errs, fmt.Errorf("failed to write action entry to remote storage: %w", err))
		}
	}
	return errors.Join(errs...)
}

// WriteOutput implements StorageBackend.
func (l *layeredStorageBackend) WriteOutput(a ActionEntry, body io.Reader) (string, error) {
	if l.localW == nil {
		panic(ErrReadonlyStorage)
	}
	if l.remoteW == nil {
		return l.localW.WriteOutput(a, body)
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
		defer bodyCopyW.Close()         // nolint:errcheck // else pipe won't know we're done
		defer io.Copy(io.Discard, body) // nolint:errcheck // drain the body to avoid deadlock
		if localPath, err1 = l.localW.WriteOutput(a, body); err1 != nil {
			err1 = fmt.Errorf("failed to write output to local storage: %w", err1)
		}
	}()
	go func() {
		defer wg.Done()
		defer io.Copy(io.Discard, bodyCopyR) // nolint:errcheck // drain the body to avoid deadlock
		if _, err2 = l.remoteW.WriteOutput(a, bodyCopyR); err2 != nil {
			err2 = fmt.Errorf("failed to write output to remote storage: %w", err2)
		}
	}()
	wg.Wait()
	return localPath, errors.Join(err1, err2)
}

func (l *layeredStorageBackend) OpenOutputFile(a ActionEntry) (io.ReadCloser, error) {
	// try local first
	f, localErr := l.local.OpenOutputFile(a)
	if localErr == nil {
		return f, nil
	}
	// if not found, try remote
	if f, err := l.remote.OpenOutputFile(a); err == nil {
		return f, nil
	}
	return nil, localErr
}

type readonlyLayeredStorageBackend layeredStorageBackend

// CheckOutputFile implements ReadonlyStorageBackend.
func (r *readonlyLayeredStorageBackend) CheckOutputFile(a ActionEntry) (string, error) {
	return (*layeredStorageBackend)(r).CheckOutputFile(a)
}

// Close implements ReadonlyStorageBackend.
func (r *readonlyLayeredStorageBackend) Close() error {
	return (*layeredStorageBackend)(r).Close()
}

// OpenOutputFile implements ReadonlyStorageBackend.
func (r *readonlyLayeredStorageBackend) OpenOutputFile(a ActionEntry) (io.ReadCloser, error) {
	return (*layeredStorageBackend)(r).OpenOutputFile(a)
}

// ReadActionEntry implements ReadonlyStorageBackend.
func (r *readonlyLayeredStorageBackend) ReadActionEntry(id []byte) (*ActionEntry, error) {
	return (*layeredStorageBackend)(r).ReadActionEntry(id)
}
