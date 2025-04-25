package resource

import "context"

type Resource interface {
	ID() string
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Ready(ctx context.Context) (bool, error) // TODO: provide not-ready details
}
