package stack

import (
	"context"
	"fmt"

	"fastcat.org/go/gdev/internal"
	"fastcat.org/go/gdev/service"
)

type Context struct {
	context.Context
	serviceModes map[string]service.Mode
}

func NewContext(
	ctx context.Context,
	opts ...ctxOption,
) *Context {
	internal.CheckLockedDown()
	return &Context{
		Context:      ctx,
		serviceModes: make(map[string]service.Mode, len(allServices)),
	}
}

type ctxOption func(*Context)

func WithServiceModes(modes map[string]service.Mode) ctxOption {
	return func(ctx *Context) {
		for svc, mode := range modes {
			if _, ok := allServices[svc]; !ok {
				panic(fmt.Errorf("service %s not registered", svc))
			}
			if !mode.Valid() {
				panic(fmt.Errorf("invalid mode %s for service %s", mode, svc))
			}
			ctx.serviceModes[svc] = mode
		}
	}
}

type modesKey struct{}

func (ctx *Context) Value(key any) any {
	if _, ok := key.(modesKey); ok {
		return ctx.serviceModes
	}
	return ctx.Context.Value(key)
}

func ServiceMode(ctx context.Context, svc string) (service.Mode, bool) {
	modes, _ := ctx.Value(modesKey{}).(map[string]service.Mode)
	if mode, ok := modes[svc]; ok {
		return mode, true
	}
	return service.ModeDefault, false
}
