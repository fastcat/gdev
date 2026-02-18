package resource

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"fastcat.org/go/gdev/addons/pm/api"
	"fastcat.org/go/gdev/lib/httpx"
	"fastcat.org/go/gdev/resource"
)

type PM struct {
	Name          string
	Config        func(context.Context) (*api.Child, error)
	LimitRestarts bool
	WaitOnStart   bool
}

func PMStatic(config api.Child) *PM {
	return &PM{
		Name:   config.Name,
		Config: func(context.Context) (*api.Child, error) { return &config, nil },
	}
}

func PMStaticInfra(config api.Child) *PM {
	return &PM{
		Name:          config.Name,
		Config:        func(context.Context) (*api.Child, error) { return &config, nil },
		LimitRestarts: true,
	}
}

func PMDynamic(name string, config func(context.Context) (*api.Child, error)) *PM {
	return &PM{
		Name:   name,
		Config: config,
	}
}

func (p *PM) WithWaitOnStart() *PM {
	p.WaitOnStart = true
	return p
}

// ID implements Resource.
func (p *PM) ID() string {
	return "pm/" + p.Name
}

// Start implements Resource.
func (p *PM) Start(ctx context.Context) error {
	client := resource.ContextValue[api.API](ctx)
	child, err := p.Config(ctx)
	if err != nil {
		return fmt.Errorf("failed to get child config: %w", err)
	}
	cur, err := client.Child(ctx, child.Name)
	if err != nil && !httpx.IsNotFound(err) {
		return fmt.Errorf("failed checking child %s status: %w", child.Name, err)
	}
	// decide if we should stop & delete the child before recreating it
	update, start := true, true
	clear := cur != nil
	if p.LimitRestarts &&
		cur != nil &&
		cur.Status.State == api.ChildRunning &&
		reflect.DeepEqual(child, &cur.Child) {
		clear, update, start = false, false, false
	}
	if clear {
		if cur.Status.State != api.ChildStopped {
			if _, err = client.StopChild(ctx, child.Name); err != nil {
				return fmt.Errorf("failed stopping child %s: %w", child.Name, err)
			}
		}
		if _, err = client.DeleteChild(ctx, child.Name); err != nil {
			return fmt.Errorf("failed deleting child %s: %w", child.Name, err)
		}
	}
	if update {
		if _, err = client.PutChild(ctx, *child); err != nil {
			return fmt.Errorf("failed adding child %s: %w", child.Name, err)
		}
	}
	if start {
		if cur, err = client.StartChild(ctx, child.Name); err != nil {
			return fmt.Errorf("failed starting child %s: %w", child.Name, err)
		}
	}
	// TODO: logging or something

	if p.WaitOnStart {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		first := true
		for {
			if first {
				fmt.Printf("Waiting for child %s to be ready...\n", child.Name)
				first = false
			}
			if ready, err := p.isReady(child, cur); err != nil {
				return fmt.Errorf("child %s failed: %w", child.Name, err)
			} else if ready {
				break
			}
			select {
			case <-ctx.Done():
				return context.Cause(ctx)
			case <-ticker.C:
				cur, err = client.Child(ctx, child.Name)
				if err != nil {
					return fmt.Errorf("failed checking child %s status: %w", child.Name, err)
				}
			}
		}
	}

	return nil
}

// Stop implements Resource.
func (p *PM) Stop(ctx context.Context) error {
	client := resource.ContextValue[api.API](ctx)
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
			if httpx.IsNotFound(err) {
				// not defined => definitely stopped
				return nil
			}
			return fmt.Errorf("failed checking child %s status: %w", child.Name, err)
		}
		switch cur.Status.State {
		case api.ChildStopped, api.ChildDone:
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
func (p *PM) Ready(ctx context.Context) (bool, error) {
	client := resource.ContextValue[api.API](ctx)
	child, err := p.Config(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to get child config: %w", err)
	}
	cur, err := client.Child(ctx, child.Name)
	if err != nil {
		return false, fmt.Errorf("failed checking child %s status: %w", child.Name, err)
	}
	return p.isReady(child, cur)
}

func (p *PM) isReady(_ *api.Child, cur *api.ChildWithStatus) (bool, error) {
	if cur.OneShot {
		// one-shots are only ready once they complete, health checks are not
		// relevant for them
		switch cur.Status.State {
		case api.ChildError, api.ChildInitError:
			return false, fmt.Errorf("one-shot child %s in error state: %s",
				cur.Name, cur.Status.State,
			)
		case api.ChildDone:
			return true, nil
		default:
			return false, nil
		}
	} else if cur.NoRestart {
		// no-restart are similar to one-shot
		switch cur.Status.State {
		case api.ChildError, api.ChildInitError:
			return false, fmt.Errorf("no-restart child %s in error state: %s",
				cur.Name, cur.Status.State,
			)
		case api.ChildDone, api.ChildStopped:
			return false, fmt.Errorf("no-restart child %s exited unexpectedly: %s",
				cur.Name, cur.Status.State,
			)
		case api.ChildRunning:
			return true, nil
		default:
			return false, nil
		}
	}

	if cur.Status.State != api.ChildRunning {
		return false, nil
	}

	if cur.HealthCheck == nil {
		// TODO: wait for it to run for a min amount of time?
		return true, nil
	}

	return cur.Status.Health.Healthy, nil
}
