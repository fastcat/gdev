package diags

import (
	"context"
	"io"
)

// A Collector represents the destination for data collected during a diags
// collection.
type Collector interface {
	// Begin is called at the start of a diags collection, before any
	// sources start collecting.
	Begin(ctx context.Context) error

	// Collect stores the contents of the provided reader under the given name.
	//
	// It is safe to call Collect concurrently from multiple goroutines, but the
	// implementation may block and only process one at a time.
	Collect(
		ctx context.Context,
		name string,
		contents io.Reader,
	) error

	// AddError may be used by Sources to note non-fatal collection errors. The
	// collector should accumulate these and store them in the output. It may do
	// so immediately, or defer that storage until the end when Finalize is
	// called.
	//
	// Any error returned from AddError is considered fatal.
	//
	// The item identifies what the Source was trying to collect when it
	// encountered the error. Typically it would be the name it would have passed
	// to Collect if there was no error, but it may be any non-empty string,
	// including strings not valid as filenames.
	AddError(
		ctx context.Context,
		item string,
		err error,
	) error

	// Finalize is called once all sources have completed.
	//
	// If any error was encountered during collection, it will be provided and the
	// collector may choose to adjust its output in light of that.
	//
	// Sources MUST NOT call Finalize.
	Finalize(ctx context.Context, collectErr error) error

	// Destination returns a human-readable description of where the collected
	// data is being stored.
	//
	// Common values might be a file path or URL from which the collected data can
	// be retrieved.
	Destination() string
}
