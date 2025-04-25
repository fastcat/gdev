package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"fastcat.org/go/gdev/pm/api"
	"fastcat.org/go/gdev/pm/internal"
)

type daemon struct {
	mu          sync.Mutex
	children    map[string]*child
	onTerminate context.CancelFunc
}

func NewDaemon() *daemon {
	return &daemon{
		children: make(map[string]*child),
	}
}

var _ api.API = (*daemon)(nil)

// child safely fetches a child from the map under the mutex
func (d *daemon) child(name string) *child {
	d.mu.Lock()
	c := d.children[name]
	d.mu.Unlock()
	return c
}

// Ping implements api.API.
func (d *daemon) Ping(ctx context.Context) error {
	// TODO: check things here?
	return nil
}

// Child implements api.API.
func (d *daemon) Child(ctx context.Context, name string) (*api.ChildWithStatus, error) {
	c := d.child(name)
	if c == nil {
		return nil, internal.WithStatus(http.StatusNotFound, fmt.Errorf("child %s not found", name))
	}
	return &api.ChildWithStatus{
		Child:  c.def,
		Status: c.Status(),
	}, nil
}

// DeleteChild implements api.API.
func (d *daemon) DeleteChild(ctx context.Context, name string) (*api.ChildWithStatus, error) {
	c := d.child(name)
	if c == nil {
		return nil, internal.WithStatus(http.StatusNotFound, fmt.Errorf("child %s not found", name))
	}
	s := c.Status()
	switch s.State {
	case api.ChildError, api.ChildInitError, api.ChildStopped:
		// ok
	default:
		return nil, internal.WithStatus(
			http.StatusPreconditionFailed,
			fmt.Errorf("cannot delete child %s in active state %s", name, s.State),
		)
	}
	c.cmds <- childDelete
	// wish we could wait under context
	c.Wait()
	return &api.ChildWithStatus{Child: c.def, Status: c.Status()}, nil
}

// PutChild implements api.API.
func (d *daemon) PutChild(ctx context.Context, child api.Child) (*api.ChildWithStatus, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if _, ok := d.children[child.Name]; ok {
		return nil, internal.WithStatus(http.StatusConflict, fmt.Errorf("child %s already exists", child.Name))
	}
	c := newChild(child)
	d.children[child.Name] = c
	go func() {
		c.run()
		d.mu.Lock()
		delete(d.children, child.Name)
		d.mu.Unlock()
	}()
	// ensure the manager goroutine has started
	c.cmds <- childPing
	return &api.ChildWithStatus{
		Child:  child,
		Status: c.Status(),
	}, nil
}

// StartChild implements api.API.
func (d *daemon) StartChild(ctx context.Context, name string) (*api.ChildWithStatus, error) {
	c := d.child(name)
	if c == nil {
		return nil, internal.WithStatus(http.StatusNotFound, fmt.Errorf("child %s not found", name))
	}
	s := c.Status()
	switch s.State {
	case api.ChildError, api.ChildInitError, api.ChildStopped:
		// ok
	case api.ChildInitRunning, api.ChildRunning:
		return nil, internal.WithStatus(
			http.StatusPreconditionFailed,
			fmt.Errorf("child %s already running (%s)", name, s.State),
		)
	default:
		return nil, internal.WithStatus(
			http.StatusPreconditionFailed,
			fmt.Errorf("cannot start child %s from state %s", name, s.State),
		)
	}
	c.cmds <- childStart
	c.cmds <- childPing // sync so we get the started status
	return &api.ChildWithStatus{Child: c.def, Status: c.Status()}, nil
}

// StopChild implements api.API.
func (d *daemon) StopChild(ctx context.Context, name string) (*api.ChildWithStatus, error) {
	c := d.child(name)
	if c == nil {
		return nil, internal.WithStatus(http.StatusNotFound, fmt.Errorf("child %s not found", name))
	}
	s := c.Status()
	switch s.State {
	case api.ChildInitRunning, api.ChildRunning:
		// ok
	case api.ChildError, api.ChildInitError:
		// also ok
	case api.ChildStopped:
		return nil, internal.WithStatus(
			http.StatusPreconditionFailed,
			fmt.Errorf("child %s already stopped", name),
		)
	case api.ChildStopping:
		return nil, internal.WithStatus(
			http.StatusPreconditionFailed,
			fmt.Errorf("child %s already stopping", name),
		)
	default:
		return nil, internal.WithStatus(
			http.StatusPreconditionFailed,
			fmt.Errorf("cannot stop child %s from state %s", name, s.State),
		)
	}
	c.cmds <- childStop
	// wait for it to stop
	t := time.NewTicker(10 * time.Millisecond)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case c.cmds <- childPing:
		}
		if c.Status().State == api.ChildStopped {
			break
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-t.C:
		}
	}
	return &api.ChildWithStatus{Child: c.def, Status: c.Status()}, nil
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

func (d *daemon) Terminate(context.Context) error {
	if d.onTerminate != nil {
		d.onTerminate()
	}
	d.mu.Lock()
	if len(d.children) == 0 {
		d.mu.Unlock()
		return nil
	}
	log.Print("terminating pm children")
	children := make([]*child, 0, len(d.children))
	for _, v := range d.children {
		children = append(children, v)
	}
	clear(d.children)
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
	log.Print("daemon done")
	return nil
}
