package resource

import (
	"context"
	"fmt"
	"time"

	"fastcat.org/go/gdev/pm/api"
)

type PM struct {
	Name   string
	Config func(context.Context) (*api.Child, error)
}

func PMStatic(name string, config api.Child) *PM {
	return &PM{name, func(context.Context) (*api.Child, error) { return &config, nil }}
}

func PMDynamic(name string, config func(context.Context) (*api.Child, error)) *PM {
	return &PM{name, config}
}

var _ Resource = (*PM)(nil)

// ID implements Resource.
func (p *PM) ID() string {
	return "pm/" + p.Name
}

// Start implements Resource.
func (p *PM) Start(ctx *Context) error {
	client := ContextValue[api.API](ctx)
	child, err := p.Config(ctx)
	if err != nil {
		return fmt.Errorf("failed to get child config: %w", err)
	}
	cur, err := client.Child(ctx, child.Name)
	if err != nil {
		if !api.IsNotFound(err) {
			return fmt.Errorf("failed checking child %s status: %w", child.Name, err)
		}
	} else {
		// stop & delete the child before recreating it
		if cur.Status.State != api.ChildStopped {
			if _, err = client.StopChild(ctx, child.Name); err != nil {
				return fmt.Errorf("failed stopping child %s: %w", child.Name, err)
			}
		}
		if _, err = client.DeleteChild(ctx, child.Name); err != nil {
			return fmt.Errorf("failed deleting child %s: %w", child.Name, err)
		}
	}
	if _, err = client.PutChild(ctx, *child); err != nil {
		return fmt.Errorf("failed adding child %s: %w", child.Name, err)
	}
	if cur, err = client.StartChild(ctx, child.Name); err != nil {
		return fmt.Errorf("failed starting child %s: %w", child.Name, err)
	}
	// TODO: logging or something
	_ = cur
	return nil
}

// Stop implements Resource.
func (p *PM) Stop(ctx *Context) error {
	client := ContextValue[api.API](ctx)
	child, err := p.Config(ctx)
	if err != nil {
		return fmt.Errorf("failed to get child config: %w", err)
	}
	cur, err := client.Child(ctx, child.Name)
	// loop until we can stop and remove the child
	retry := time.NewTicker(10 * time.Millisecond)
	defer retry.Stop()
	for {
		if err != nil {
			if api.IsNotFound(err) {
				// not defined => definitely stopped
				return nil
			}
			return fmt.Errorf("failed checking child %s status: %w", child.Name, err)
		}
		switch cur.Status.State {
		case api.ChildStopped:
			cur, err = client.DeleteChild(ctx, child.Name)
			// check cur/err again at the top
		case api.ChildError, api.ChildInitError, api.ChildInitRunning, api.ChildRunning:
			cur, err = client.StopChild(ctx, child.Name)
		case api.ChildStopping:
			// wait
			select {
			case <-ctx.Done():
				return context.Cause(ctx)
			case <-retry.C:
				// refresh and loop back to check status again
				cur, err = client.Child(ctx, child.Name)
			}
		default:
			return fmt.Errorf("child %s: unrecognized state %q", child.Name, cur.Status.State)
		}
	}
}

// Ready implements Resource
func (p *PM) Ready(ctx *Context) (bool, error) {
	client := ContextValue[api.API](ctx)
	child, err := p.Config(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to get child config: %w", err)
	}
	cur, err := client.Child(ctx, child.Name)
	if err != nil {
		return false, fmt.Errorf("failed checking child %s status: %w", child.Name, err)
	}
	if cur.Status.State != api.ChildRunning {
		// TODO: Done state for one-shot jobs
		// TODO: say why it's unhealthy
		return false, nil
	}
	if child.HealthCheck == nil {
		// TODO: wait for it to run for a min amount of time?
		return true, nil
	}
	return cur.Status.Health.Healthy, nil
}
