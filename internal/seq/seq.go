// Package seq provides a definition of an internal Seq interface for
// use across packages. It is used to avoid circular dependencies.
package seq

type Seq interface {
	First() any
	Next() Seq
}
