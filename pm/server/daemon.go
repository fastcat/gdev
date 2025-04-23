package server

import (
	"context"
	"fmt"
	"sync"

	"fastcat.org/go/gdev/pm/api"
)

type daemon struct {
	mu       sync.Mutex
	children map[string]*child
}

func NewDaemon() *daemon {
	return &daemon{children: make(map[string]*child)}
}

var _ api.API = (*daemon)(nil)

// Ping implements api.API.
func (d *daemon) Ping(ctx context.Context) error {
	// TODO: check things here?
	return nil
}

// Child implements api.API.
func (d *daemon) Child(ctx context.Context, name string) (*api.ChildWithStatus, error) {
	panic("unimplemented")
}

// DeleteChild implements api.API.
func (d *daemon) DeleteChild(ctx context.Context, name string) (*api.ChildWithStatus, error) {
	panic("unimplemented")
}

// PutChild implements api.API.
func (d *daemon) PutChild(ctx context.Context, child api.Child) (*api.ChildWithStatus, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if _, ok := d.children[child.Name]; ok {
		// TODO: http code
		return nil, fmt.Errorf("child %s already exists", child.Name)
	}
	c := newChild(child)
	d.children[child.Name] = c
	go c.run()
	// ensure it's going
	c.cmds <- childPing
	return &api.ChildWithStatus{
		Child:  child,
		Status: c.Status(),
	}, nil
}

// StartChild implements api.API.
func (d *daemon) StartChild(ctx context.Context, name string) (*api.ChildWithStatus, error) {
	panic("unimplemented")
}

// StopChild implements api.API.
func (d *daemon) StopChild(ctx context.Context, name string) (*api.ChildWithStatus, error) {
	panic("unimplemented")
}

// Summary implements api.API.
func (d *daemon) Summary(ctx context.Context) ([]api.ChildSummary, error) {
	panic("unimplemented")
}
