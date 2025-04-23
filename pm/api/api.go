package api

import "context"

type API interface {
	Ping(ctx context.Context) error
	Summary(ctx context.Context) ([]ChildSummary, error)
	Child(ctx context.Context, name string) (ChildWithStatus, error)
	PutChild(ctx context.Context, child Child) (ChildWithStatus, error)
	StartChild(ctx context.Context, name string) (ChildWithStatus, error)
	StopChild(ctx context.Context, name string) (ChildWithStatus, error)
	DeleteChild(ctx context.Context, name string) (ChildWithStatus, error)
	// TODO: ChildLogs
}
