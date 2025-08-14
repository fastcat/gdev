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

func NewContext(ctx context.Context) *Context {
	return internal.NewContext(ctx)
}

func NewKey[T any](name string) InfoKey[T] {
	return internal.NewKey[T](name)
}

func Save[T any](ctx *Context, k InfoKey[T], v T) {
	internal.Save(ctx, k, v)
}

func Set[T any](ctx *Context, k InfoKey[T], v T) {
	internal.Set(ctx, k, v)
}

func Get[T any](ctx *Context, k InfoKey[T]) (T, bool) {
	return internal.Get(ctx, k)
}

func Clear[T any](ctx *Context, k InfoKey[T]) {
	internal.Clear(ctx, k)
}
