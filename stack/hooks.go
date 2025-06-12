package stack

import (
	"context"
	"fmt"

	"fastcat.org/go/gdev/internal"
	"fastcat.org/go/gdev/service"
)

type PreStartServiceHook interface {
	Name() string
	Run(context.Context, service.Service) error
}

type preStartServiceHook struct {
	name string
	run  func(context.Context, service.Service) error
}

func (h *preStartServiceHook) Name() string { return h.name }
func (h *preStartServiceHook) Run(ctx context.Context, svc service.Service) error {
	return h.run(ctx, svc)
}

type PreStartHook interface {
	Name() string
	Run(ctx context.Context, infra, svcs []service.Service) error
}

type preStartHook struct {
	name string
	run  func(ctx context.Context, infra, svcs []service.Service) error
}

func (h *preStartHook) Name() string { return h.name }
func (h *preStartHook) Run(ctx context.Context, infra, svcs []service.Service) error {
	return h.run(ctx, infra, svcs)
}

var (
	preStartServiceHooks []PreStartServiceHook
	preStartHooks        []PreStartHook
)

func AddPreStartServiceHook(name string, run func(context.Context, service.Service) error) {
	internal.CheckCanCustomize()
	if name == "" {
		panic("pre-start service hook name must not be empty")
	}
	if run == nil {
		panic("pre-start service hook function must not be nil")
	}
	for _, h := range preStartServiceHooks {
		if h.Name() == name {
			panic("pre-start service hook with name " + name + " already exists")
		}
	}
	preStartServiceHooks = append(preStartServiceHooks, &preStartServiceHook{name: name, run: run})
}

func AddPreStartHook(name string, run func(ctx context.Context, infra, svcs []service.Service) error) {
	internal.CheckCanCustomize()
	if name == "" {
		panic("pre-start hook name must not be empty")
	}
	if run == nil {
		panic("pre-start hook function must not be nil")
	}
	for _, h := range preStartHooks {
		if h.Name() == name {
			panic("pre-start hook with name " + name + " already exists")
		}
	}
	preStartHooks = append(preStartHooks, &preStartHook{name: name, run: run})
}

func preStart(ctx context.Context, infra, svcs []service.Service) error {
	internal.CheckLockedDown()
	for _, svc := range svcs {
		for _, hook := range preStartServiceHooks {
			if err := hook.Run(ctx, svc); err != nil {
				return fmt.Errorf("error running pre-start hook %s for service %s: %w", hook.Name(), svc.Name(), err)
			}
		}
	}
	for _, hook := range preStartHooks {
		if err := hook.Run(ctx, infra, svcs); err != nil {
			return fmt.Errorf("error running pre-start hook %s: %w", hook.Name(), err)
		}
	}
	return nil
}
