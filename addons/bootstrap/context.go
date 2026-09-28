package bootstrap

import (
	"context"

	"fastcat.org/go/gdev/addons/bootstrap/internal"
)

// aliases exposing a limited subset of the internal Context API

type (
	Context        = internal.Context
	InfoKey[T any] = internal.InfoKey[T]
)

func NewContextWithDefaults(ctx context.Context) *Context {
	return internal.NewContextWithDefaults(ctx)
}

func NewKey[T any](name string) InfoKey[T] {
	return internal.NewKey[T](name)
}

// Deprecated: modernize
//
//go:fix inline
func Save[T any](ctx *Context, k InfoKey[T], v T) {
	ctx.Save(k, v)
}

// Deprecated: modernize
//
//go:fix inline
func Set[T any](ctx *Context, k InfoKey[T], v T) {
	ctx.Set(k, v)
}

// Deprecated: modernize
//
//go:fix inline
func Get[T any](ctx *Context, k InfoKey[T]) (T, bool) {
	return ctx.Get(k)
}

// Deprecated: modernize
//
//go:fix inline
func Clear[T any](ctx *Context, k InfoKey[T]) {
	ctx.Clear(k)
}
