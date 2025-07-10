package gocache_http

import (
	"errors"
	"io"
	"net/http"
)

type writer struct {
	c     *http.Client
	req   *http.Request
	resp  *http.Response
	run   chan struct{}
	errCh chan error
	pw    *io.PipeWriter
}

func (w *writer) start() {
	w.run = make(chan struct{})
	w.errCh = make(chan error, 1)
	go w.do()
	// wait for it to be ready, it won't be safe to call Close() until this
	<-w.run
}

func (w *writer) do() {
	run, errCh := w.run, w.errCh
	defer close(errCh)
	// tell start we have read all the race-prone members we need
	run <- struct{}{}

	resp, err := w.c.Do(w.req)
	if err != nil {
		errCh <- err
		return
	}
	w.resp = resp
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
			errs = append(errs, err)
		}
	}
	// we now know the background goroutine has finished, so we don't need to
	// worry about races for the rest of this
	if w.resp != nil {
		// this can't return a meaningful error
		_, _ = io.Copy(io.Discard, w.resp.Body)
		_ = w.resp.Body.Close()
		w.resp = nil
	}
	// leave errCh alone, it'll be closed by now
	return errors.Join(errs...)
}

// Sync implements gocache.WriteFile.
//
// It is a no-op
func (w *writer) Sync() error { return nil }

// Write implements gocache.WriteFile.
func (w *writer) Write(data []byte) (n int, err error) {
	return w.pw.Write(data)
}
