package server

import (
	"context"

	"fastcat.org/go/gdev/pm/api"
)

type daemon struct {
	// TODO
}

var _ api.API = (*daemon)(nil)

// Ping implements api.API.
func (d *daemon) Ping(ctx context.Context) error {
	// TODO: check things here?
	return nil
}

// Child implements api.API.
func (d *daemon) Child(ctx context.Context, name string) (api.ChildWithStatus, error) {
	panic("unimplemented")
}

// DeleteChild implements api.API.
func (d *daemon) DeleteChild(ctx context.Context, name string) (api.ChildWithStatus, error) {
	panic("unimplemented")
}

// PutChild implements api.API.
func (d *daemon) PutChild(ctx context.Context, child api.Child) (api.ChildWithStatus, error) {
	panic("unimplemented")
}

// StartChild implements api.API.
func (d *daemon) StartChild(ctx context.Context, name string) (api.ChildWithStatus, error) {
	panic("unimplemented")
}

// StopChild implements api.API.
func (d *daemon) StopChild(ctx context.Context, name string) (api.ChildWithStatus, error) {
	panic("unimplemented")
}

// Summary implements api.API.
func (d *daemon) Summary(ctx context.Context) ([]api.ChildSummary, error) {
	panic("unimplemented")
}
