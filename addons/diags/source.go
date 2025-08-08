package diags

import "context"

// A Source represents a source of data to be captured as part of a diags
// collection.
//
// It should make one or more calls to the provided Collector to store the data
// it provides. It can and should do its work concurrently as much as is
// possible.
//
// See: [Collector]
type Source interface {
	Collect(
		ctx context.Context,
		collector Collector,
	) error
}

type SourceFunc func(
	ctx context.Context,
	collector Collector,
) error

func (f SourceFunc) Collect(
	ctx context.Context,
	collector Collector,
) error {
	return f(ctx, collector)
}
