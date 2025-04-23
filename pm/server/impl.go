package server

import (
	"context"

	"fastcat.org/go/gdev/pm/api"
)

type server struct {
	// TODO
}

var _ api.API = (*server)(nil)

// Ping implements api.API.
func (s *server) Ping(ctx context.Context) error {
	// TODO: check things here?
	return nil
}

// Child implements api.API.
func (s *server) Child(ctx context.Context, name string) (api.ChildWithStatus, error) {
	panic("unimplemented")
}

// DeleteChild implements api.API.
func (s *server) DeleteChild(ctx context.Context, name string) (api.ChildWithStatus, error) {
	panic("unimplemented")
}

// PutChild implements api.API.
func (s *server) PutChild(ctx context.Context, child api.Child) (api.ChildWithStatus, error) {
	panic("unimplemented")
}

// StartChild implements api.API.
func (s *server) StartChild(ctx context.Context, name string) (api.ChildWithStatus, error) {
	panic("unimplemented")
}

// StopChild implements api.API.
func (s *server) StopChild(ctx context.Context, name string) (api.ChildWithStatus, error) {
	panic("unimplemented")
}

// Summary implements api.API.
func (s *server) Summary(ctx context.Context) ([]api.ChildSummary, error) {
	panic("unimplemented")
}
