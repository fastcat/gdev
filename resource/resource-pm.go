package resource

import (
	"context"
	"fmt"

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
func (p *PM) Start(ctx context.Context) error {
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
func (p *PM) Stop(ctx context.Context) error {
	panic("unimplemented")
}

// Ready implements Resource
func (p *PM) Ready(ctx context.Context) (bool, error) {
	panic("unimplemented")
}
