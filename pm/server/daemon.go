package server

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

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
	go func() {
		c.run()
		d.mu.Lock()
		delete(d.children, child.Name)
		d.mu.Unlock()
	}()
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
	d.mu.Lock()
	defer d.mu.Unlock()
	ret := make([]api.ChildSummary, 0, len(d.children))
	for _, child := range d.children {
		status := child.Status()
		ret = append(ret, api.ChildSummary{
			Name:  child.def.Name,
			State: status.State,
			// TODO: find a running init container
			Pid: status.Main.Pid,
		})
	}
	return ret, nil
}

func (d *daemon) terminate() error {
	log.Print("terminating pm children")
	d.mu.Lock()
	children := make([]*child, 0, len(d.children))
	for _, v := range d.children {
		children = append(children, v)
	}
	d.mu.Unlock()
	var wg sync.WaitGroup
	for _, child := range children {
		wg.Add(1)
		go func() {
			defer wg.Done()
			child.cmds <- childStop
			// wait for it to stop
			// TODO: avoid polling
			check := time.NewTicker(10 * time.Millisecond)
			defer check.Stop()
			for range check.C {
				if s := child.Status().State; s == api.ChildError || s == api.ChildStopped {
					break
				}
			}
			child.cmds <- childDelete
			child.Wait()
		}()
	}
	wg.Wait()
	return nil
}
