package bootstrap

import (
	"context"
	"fmt"
	"reflect"
)

type Context struct {
	context.Context
	info map[infoKey]any
}

func NewContext(ctx context.Context) *Context {
	return &Context{
		Context: ctx,
		info:    map[infoKey]any{},
	}
}

type InfoKey[T any] struct {
	k string
	_ [0]*T // make keys for different types non-convertible
}

func (k InfoKey[T]) key() string       { return k.k }
func (k InfoKey[T]) typ() reflect.Type { return reflect.TypeFor[T]() }

type infoKey interface {
	key() string
	typ() reflect.Type
}

func NewKey[T any](name string) InfoKey[T] {
	return InfoKey[T]{k: name}
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
	if k, ok := key.(infoKey); ok {
		return ctx.info[k]
	}
	return ctx.Context.Value(key)
}
