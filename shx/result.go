package shx

import (
	"errors"
	"io"
	"os"
)

type Result struct {
	stdoutCapture *outCapture
	stderrCapture *outCapture
	stdinFeed     io.ReadCloser

	exitErr      error
	processState *os.ProcessState
}

func (r *Result) Err() error {
	return r.exitErr
}

func (r *Result) Close() error {
	var errs []error
	if r.stdinFeed != nil {
		errs = append(errs, r.stdinFeed.Close())
		r.stdinFeed = nil
	}
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
