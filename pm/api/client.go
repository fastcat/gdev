package api

import "context"

type Client interface {
	Summary(context.Context) ([]ChildSummary, error)
	Child(context.Context, string) (ChildWithStatus, error)
	PutChild(context.Context, Child) (ChildWithStatus, error)
	StartChild(context.Context, string) (ChildWithStatus, error)
	StopChild(context.Context, string) (ChildWithStatus, error)
	DeleteChild(context.Context, string) (ChildWithStatus, error)
	// TODO: ChildLogs
}
