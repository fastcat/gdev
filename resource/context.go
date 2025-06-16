package resource

import (
	"context"
	"fmt"
	"reflect"
)

type ctxKeyVal[T any] struct {
	_ [0]*T // make sure these aren't convertible between different T
}

func (k ctxKeyVal[T]) typ() reflect.Type {
	return reflect.TypeFor[T]()
}

type ctxKey interface {
	typ() reflect.Type
}

type ctxEntry struct {
	initializer func(context.Context) (any, error)
}

var ctxEntries = map[ctxKey]ctxEntry{}

func AddContextEntry[T any](initializer func(context.Context) (T, error)) {
	key := ctxKeyVal[T]{}
	if _, ok := ctxEntries[key]; ok {
		panic(fmt.Errorf("already registered for type %v", reflect.TypeFor[T]()))
	}
	anyInitializer := func(ctx context.Context) (any, error) { return initializer(ctx) }
	ctxEntries[key] = ctxEntry{anyInitializer}
}

type dryRunKey struct{}

type Context struct {
	context.Context
	entries map[ctxKey]any
	dryRun  bool
}

type ContextOption func(*Context)

func WithDryRun() ContextOption {
	return func(ctx *Context) {
		ctx.dryRun = true
	}
}

func NewContext(ctx context.Context, opts ...ContextOption) (*Context, error) {
	rc := NewEmptyContext(ctx, opts...)
	for k, e := range ctxEntries {
		if v, err := e.initializer(rc); err != nil {
			return nil, fmt.Errorf("failed to initialize %v: %w", k.typ(), err)
		} else {
			rc.entries[k] = v
		}
	}
	return rc, nil
}

func NewEmptyContext(ctx context.Context, opts ...ContextOption) *Context {
	rc := &Context{
		ctx,
		make(map[ctxKey]any, len(ctxEntries)),
		false,
	}
	for _, opt := range opts {
		opt(rc)
	}
	return rc
}

func (ctx *Context) Value(key any) any {
	if _, ok := key.(dryRunKey); ok {
		return ctx.dryRun
	}
	if ck, ok := key.(ctxKey); ok {
		if val, ok := ctx.entries[ck]; ok {
			return val
		}
		// delayed instantiation
		if e, ok := ctxEntries[ck]; ok {
			if v, err := e.initializer(ctx); err != nil {
				// oops, too late to exit gracefully!
				panic(err)
			} else {
				ctx.entries[ck] = v
				return v
			}
		}
	}
	return ctx.Context.Value(key)
}

func ContextValue[T any](ctx context.Context) T {
	key := ctxKeyVal[T]{}
	if _, ok := ctxEntries[key]; !ok {
		panic(fmt.Errorf("type %v not registered", key.typ()))
	}
	if rc, ok := ctx.(*Context); ok {
		val, ok := rc.entries[key]
		if !ok {
			// delayed instantiation
			if e, ok := ctxEntries[key]; ok {
				if val, err := e.initializer(ctx); err != nil {
					// oops, too late to exit gracefully!
					panic(err)
				} else {
					rc.entries[key] = val
					return val.(T)
				}
			}
			panic(fmt.Errorf("type %v not initialized", key.typ()))
		}
		return val.(T)
	}
	return ctx.Value(key).(T)
}

func DryRun(ctx context.Context) bool {
	if rc, ok := ctx.(*Context); ok {
		return rc.dryRun
	}
	if v := ctx.Value(dryRunKey{}); v != nil {
		return v.(bool)
	}
	return false
}
