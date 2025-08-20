package shx

import (
	"errors"
	"io"
	"os"
)

type Result struct {
	stdoutCapture *outCapture
	stderrCapture *outCapture

	exitErr      error
	processState *os.ProcessState
}

func (r *Result) Err() error {
	return r.exitErr
}

// Close releases any resources associated with the result of running a command.
//
// If no output capture was enabled, it is safe to skip calling this.
func (r *Result) Close() error {
	if r == nil {
		return nil
	}
	var errs []error
	if r.stdoutCapture != nil {
		errs = append(errs, r.stdoutCapture.Close())
		r.stdoutCapture = nil
	}
	if r.stderrCapture != nil {
		errs = append(errs, r.stderrCapture.Close())
		r.stderrCapture = nil
	}
	return errors.Join(errs...)
}

func (r *Result) execDone() error {
	var errs []error
	if r.stdoutCapture != nil {
		if err := r.stdoutCapture.doneWriting(); err != nil {
			errs = append(errs, err)
		}
	}
	if r.stderrCapture != nil {
		if err := r.stderrCapture.doneWriting(); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// Stdout returns a reader over the captured stdout output.
//
// If stdout was not captured, or was streamed to a custom Writer, this returns
// nil.
func (r *Result) Stdout() io.Reader {
	return r.stdoutCapture.reader()
}

// Stderr returns a reader over the captured stderr output.
//
// If stderr was not captured, or was streamed to a custom Writer, this returns
// nil.
func (r *Result) Stderr() io.Reader {
	return r.stderrCapture.reader()
}
