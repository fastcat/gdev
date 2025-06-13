package service

import (
	"context"
	"fmt"

	"fastcat.org/go/gdev/internal"
)

type Context struct {
	context.Context
	serviceModes map[string]Mode
}

func NewContext(
	ctx context.Context,
	opts ...ContextOption,
) *Context {
	internal.CheckLockedDown()
	return &Context{
		Context:      ctx,
		serviceModes: make(map[string]Mode),
	}
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

type modesKey struct{}

func (ctx *Context) Value(key any) any {
	if _, ok := key.(modesKey); ok {
		return ctx.serviceModes
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
