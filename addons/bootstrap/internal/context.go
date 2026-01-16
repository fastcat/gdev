package internal

import (
	"context"
	"fmt"
	"maps"
)

type Context struct {
	context.Context
	info map[AnyInfoKey]any
}

var defaults = map[AnyInfoKey]any{}

// NewContextWithDefaults creates a new Context with default InfoKey values as
// configured with SetDefault.
func NewContextWithDefaults(ctx context.Context) *Context {
	bCtx := NewEmptyContext(ctx)
	maps.Copy(bCtx.info, defaults)
	return bCtx
}

// NewEmptyContext creates a new empty Context without any default InfoKey
// values set.
func NewEmptyContext(ctx context.Context) *Context {
	return &Context{
		Context: ctx,
		info:    map[AnyInfoKey]any{},
	}
}

// Save stores a value, but only if it is not already set.
//
// If the value is already set, it panics.
func Save[T any](ctx *Context, k InfoKey[T], v T) {
	if _, ok := ctx.info[k]; ok {
		panic(fmt.Errorf("already saved %s for %v", k.k, k.typ()))
	}
	ctx.info[k] = v
}

// Set is like save, but it will overwrite any existing value as well.
func Set[T any](ctx *Context, k InfoKey[T], v T) {
	ctx.info[k] = v
}

func SetDefault[T any](k InfoKey[T], v T) {
	if _, ok := defaults[k]; ok {
		panic(fmt.Errorf("already saved default %s for %v", k.k, k.typ()))
	}
	defaults[k] = v
}

func Get[T any](ctx *Context, k InfoKey[T]) (T, bool) {
	v, ok := ctx.info[k]
	if !ok {
		var t T
		return t, ok
	}
	return v.(T), ok
}

func Clear[T any](ctx *Context, k InfoKey[T]) {
	if _, ok := ctx.info[k]; !ok {
		panic(fmt.Errorf("not saved %s for %v", k.k, k.typ()))
	}
	delete(ctx.info, k)
}

func (ctx *Context) Value(key any) any {
	if k, ok := key.(AnyInfoKey); ok {
		return ctx.info[k]
	}
	return ctx.Context.Value(key)
}
