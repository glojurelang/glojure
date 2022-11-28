package value

import (
	"context"
)

// Value is the interface that all values in the language implement.
type Value interface{}

type Sequence interface {
	First() Value
	Rest() Sequence
	IsEmpty() bool
}

// Enumerable is an interface for compound values that support
// enumeration.
type Enumerable interface {
	// Enumerate returns a channel that will yield all of the values
	// in the compound value.
	Enumerate() (values <-chan Value, cancel func())
}

// EnumerableFunc is a function that implements the Enumerable
// interface.
type EnumerableFunc func() (<-chan Value, func())

func (f EnumerableFunc) Enumerate() (<-chan Value, func()) {
	return f()
}

// EnumerateAll returns all values in the sequence. If the sequence is
// infinite, this will never return unless the context is cancelled.
func EnumerateAll(ctx context.Context, e Enumerable) ([]Value, error) {
	ch, cancel := e.Enumerate()
	defer cancel()

	var values []Value
	for {
		select {
		case v, ok := <-ch:
			if !ok {
				return values, nil
			}
			values = append(values, v)
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

// Conjer is an interface for values that can be conjed onto.
type Conjer interface {
	Value
	Conj(...Value) Conjer
}

// Counter is an interface for compound values whose elements can be
// counted.
type Counter interface {
	Count() int
}

// Nther is an interface for compound values whose elements can be
// accessed by index.
type Nther interface {
	Nth(int) (v Value, ok bool)
}

// MustNth returns the nth element of the vector. It panics if the
// index is out of range.
func MustNth(nth Nther, i int) Value {
	v, ok := nth.Nth(i)
	if !ok {
		panic("index out of range")
	}
	return v
}

// GoValuer is an interface for values that can be converted to a Go
// value.
type GoValuer interface {
	GoValue() interface{}
}
