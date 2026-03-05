package service

import (
	"context"
	"fmt"

	"fastcat.org/go/gdev/internal"
)

type Context struct {
	context.Context
	serviceModes  map[string]Mode
	noServiceWait bool
}

func NewContext(
	ctx context.Context,
	opts ...ContextOption,
) *Context {
	internal.CheckLockedDown()
	c := &Context{
		Context:      ctx,
		serviceModes: make(map[string]Mode),
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

type ContextOption func(*Context)

func WithServiceModes(modes map[string]Mode) ContextOption {
	return func(ctx *Context) {
		for svc, mode := range modes {
			// TODO: find a way to validate service names without an import cycle
			// if _, ok := allServices[svc]; !ok {
			// 	panic(fmt.Errorf("service %s not registered", svc))
			// }
			if !mode.Valid() {
				panic(fmt.Errorf("invalid mode %s for service %s", mode, svc))
			}
			ctx.serviceModes[svc] = mode
		}
	}
}

// WithoutServiceWait requests that the final wait for non-infrastructure services to be ready be skipped.
func WithoutServiceWait() ContextOption {
	return func(ctx *Context) {
		ctx.noServiceWait = true
	}
}

type (
	modesKey         struct{}
	noServiceWaitKey struct{}
)

func (ctx *Context) Value(key any) any {
	if _, ok := key.(modesKey); ok {
		return ctx.serviceModes
	} else if _, ok := key.(noServiceWaitKey); ok {
		return ctx.noServiceWait
	}
	return ctx.Context.Value(key)
}

func ServiceMode(ctx context.Context, svc string) (Mode, bool) {
	modes, _ := ctx.Value(modesKey{}).(map[string]Mode)
	if mode, ok := modes[svc]; ok {
		return mode, true
	}
	return ModeDefault, false
}

func NoServiceWait(ctx context.Context) bool {
	noServiceWait, ok := ctx.Value(noServiceWaitKey{}).(bool)
	return ok && noServiceWait
}
