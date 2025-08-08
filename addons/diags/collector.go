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
