package internal

import (
	"context"
	"fmt"
)

type Context struct {
	context.Context
	info map[AnyInfoKey]any
}

func NewContext(ctx context.Context) *Context {
	return &Context{
		Context: ctx,
		info:    map[AnyInfoKey]any{},
	}
}

func Save[T any](ctx *Context, k InfoKey[T], v T) {
	if _, ok := ctx.info[k]; ok {
		panic(fmt.Errorf("already saved %s for %v", k.k, k.typ()))
	}
	ctx.info[k] = v
}

func Get[T any](ctx *Context, k InfoKey[T]) (T, bool) {
	v, ok := ctx.info[k]
	if !ok {
		var t T
		return t, ok
	}
	return v.(T), ok
}

func (ctx *Context) Value(key any) any {
	if k, ok := key.(AnyInfoKey); ok {
		return ctx.info[k]
	}
	return ctx.Context.Value(key)
}
