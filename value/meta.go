package value

import "fmt"

// IMeta is an interface for values that can have metadata.
type IMeta interface {
	WithMeta(meta IPersistentMap) interface{}
}

// WithMeta returns a new value with the given metadata.
func WithMeta(v interface{}, meta IPersistentMap) (interface{}, error) {
	imeta, ok := v.(IMeta)
	if !ok {
		return nil, fmt.Errorf("value of type %T can't have metadata", v)
	}
	return imeta.WithMeta(meta), nil
}
