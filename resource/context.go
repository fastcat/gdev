package resource

import (
	"context"
	"fmt"
	"reflect"
)

// ctxKeyVal[T] implements [ctxKey].
type ctxKeyVal[T any] struct {
	_ [0]*T // make sure these aren't convertible between different T
}

func (k ctxKeyVal[T]) typ() reflect.Type {
	return reflect.TypeFor[T]()
}

// ctxKey is an interface for keys used in the context entries map, to store a
// value associated with a specific type, as a simplistic dependency injection
// mechanism.
//
// The only implementation of this interface is [ctxKeyVal[T]].
type ctxKey interface {
	typ() reflect.Type
}

// ctxEntry represents a dependency injection binding in the context. It is
// associated with some specific type, and thus [ctxKey].
//
// It captures an initializer function to provide the injected value.
type ctxEntry struct {
	initializer func(context.Context) (any, error)
}

var ctxEntries = map[ctxKey]ctxEntry{}

// AddContextEntry registers a new context entry for the given type T. Only one
// initializer may be present per type. Only the exact type T is registered, not
// any compatible interfaces.
//
// It is unsafe to call this function concurrently with itself, or with any
// functions/methods that create or use a [Context]. In general, this should
// only be called during app initialization.
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
	// entries captures a map from the type-specific [ctxKey] to the initialized
	// value.
	entries map[ctxKey]any
	dryRun  bool
}

type ContextOption func(*Context)

func WithDryRun() ContextOption {
	return func(ctx *Context) {
		ctx.dryRun = true
	}
}

func WithValue[T any](val T) ContextOption {
	key := ctxKeyVal[T]{}
	return func(ctx *Context) {
		if _, ok := ctx.entries[key]; ok {
			panic(fmt.Errorf("type %v already has a value in context", key.typ()))
		}
		ctx.entries[key] = val
	}
}

// NewContext creates a new [Context] with the given parent context and options.
//
// All context entries registered with [AddContextEntry] will be initialized
// proactively, and if any fail this function will return an error.
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

// NewEmptyContext creates a new [Context] with the given parent context and
// options, but does not initialize any context entries. If requested, they will
// be initialized on demand, but if they fail in that case, it will panic.
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

// ContextValue retrieves a value of type T from the context, which must have
// been registered with [AddContextEntry]. If no value or initializer for the
// type is found, it will panic. If an initializer is found but not a value, it
// will also panic.
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
