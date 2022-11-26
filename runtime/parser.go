package runtime

import (
	"io"

	"github.com/glojurelang/glojure/reader"
	"github.com/glojurelang/glojure/value"
)

type parseOptions struct {
	filename string
}

// ParseOption represents an option that can be passed to Parse.
type ParseOption func(*parseOptions)

// WithFilename sets the filename to be associated with the input.
func WithFilename(filename string) ParseOption {
	return func(o *parseOptions) {
		o.filename = filename
	}
}

func Parse(r io.RuneScanner, opts ...ParseOption) (*Program, error) {
	o := &parseOptions{}
	for _, opt := range opts {
		opt(o)
	}

	var readOpts []reader.Option
	if o.filename != "" {
		readOpts = append(readOpts, reader.WithFilename(o.filename))
	}

	rr := reader.New(r, readOpts...)
	nodes, err := rr.ReadAll()
	if err != nil {
		return nil, err
	}

	return newProgramFromValue(nodes)
}

func newProgramFromValue(values []value.Value) (*Program, error) {
	p := &Program{
		nodes: values,
	}

	return p, nil
}
