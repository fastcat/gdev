package stack

import (
	"context"
	"fmt"

	"fastcat.org/go/gdev/internal"
	"fastcat.org/go/gdev/service"
)

type PreStartHook interface {
	Name() string
	BeforeServices(ctx context.Context, infra, svcs []service.Service) error
	Service(context.Context, service.Service) error
	AfterServices(ctx context.Context, infra, svcs []service.Service) error
}

type preStartHook struct {
	name           string
	beforeServices func(ctx context.Context, infra, svcs []service.Service) error
	service        func(ctx context.Context, svc service.Service) error
	afterServices  func(ctx context.Context, infra, svcs []service.Service) error
}

func (h *preStartHook) Name() string { return h.name }

func (h *preStartHook) BeforeServices(ctx context.Context, infra, svcs []service.Service) error {
	if h.beforeServices == nil {
		return nil // No specific before services hook defined
	}
	return h.beforeServices(ctx, infra, svcs)
}

func (h *preStartHook) Service(ctx context.Context, svc service.Service) error {
	if h.service == nil {
		return nil // No specific service hook defined
	}
	return h.service(ctx, svc)
}

func (h *preStartHook) AfterServices(ctx context.Context, infra, svcs []service.Service) error {
	if h.afterServices == nil {
		return nil // No specific after services hook defined
	}
	return h.afterServices(ctx, infra, svcs)
}

var preStartHookFactories []func() PreStartHook

func AddPreStartHook(fn func() PreStartHook) {
	internal.CheckCanCustomize()
	preStartHookFactories = append(preStartHookFactories, fn)
}

// AddPreStartHookType is a helper to register a [PreStartHook] implementation
// for a type T where *T implements [PreStartHook]. When running the pre-start
// process, a new(T) will be created and used.
//
// This is useful for a type that stores state between the pre-start stages, but
// which does not need initialization before the first stage.
func AddPreStartHookType[T any, P interface {
	*T
	PreStartHook
}]() {
	AddPreStartHook(func() PreStartHook { return P(new(T)) })
}

// PreStartHookFuncs creates a PreStartHook built from the given functions. Any
// of them may be nil if no action is needed. A hook with all nil functions is
// silly, but not an error.
func PreStartHookFuncs(
	name string,
	beforeServices func(ctx context.Context, infra, svcs []service.Service) error,
	service func(ctx context.Context, svc service.Service) error,
	afterServices func(ctx context.Context, infra, svcs []service.Service) error,
) PreStartHook {
	if name == "" {
		panic("pre-start hook name must not be empty")
	}
	if beforeServices == nil && service == nil && afterServices == nil {
		panic("at least one hook function must be provided")
	}
	return &preStartHook{
		name:           name,
		beforeServices: beforeServices,
		service:        service,
		afterServices:  afterServices,
	}
}

func preStart(ctx context.Context, infra, svcs []service.Service) error {
	internal.CheckLockedDown()
	hooks := make([]PreStartHook, 0, len(preStartHookFactories))
	for _, factory := range preStartHookFactories {
		hook := factory()
		hooks = append(hooks, hook)
	}
	for _, hook := range hooks {
		if err := hook.BeforeServices(ctx, infra, svcs); err != nil {
			return fmt.Errorf("error running pre-start hook %s: %w", hook.Name(), err)
		}
	}
	for _, svc := range svcs {
		for _, hook := range hooks {
			if err := hook.Service(ctx, svc); err != nil {
				return fmt.Errorf("error running pre-start hook %s for service %s: %w", hook.Name(), svc.Name(), err)
			}
		}
	}
	for _, hook := range hooks {
		if err := hook.AfterServices(ctx, infra, svcs); err != nil {
			return fmt.Errorf("error running pre-start hook %s: %w", hook.Name(), err)
		}
	}
	return nil
}
