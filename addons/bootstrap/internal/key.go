package internal

import (
	"fmt"
	"reflect"
)

type InfoKey[T any] struct {
	k string
	_ [0]*T // make keys for different types non-convertible
}

func (k InfoKey[T]) key() string       { return k.k }
func (k InfoKey[T]) typ() reflect.Type { return reflect.TypeFor[T]() }

// infoKey is a non-generic interface implemented exclusively by [InfoKey[T]].
// It exists so that InfoKeys can be map keys.
type infoKey interface {
	key() string
	typ() reflect.Type
}

func NewKey[T any](name string) InfoKey[T] {
	return InfoKey[T]{k: name}
}

// Format implements fmt.Formatter.
func (k InfoKey[T]) Format(f fmt.State, _ rune) {
	_, _ = fmt.Fprintf(f, "%s[%s]", k.k, reflect.TypeFor[T]().Name())
}
