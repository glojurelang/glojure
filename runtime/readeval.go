package runtime

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/glojurelang/glojure/reader"
	"github.com/glojurelang/glojure/value"
)

type (
	// ReadEvalOption is an option for ReadEval.
	ReadEvalOption func(*readEvalOptions)

	readEvalOptions struct {
		// env is the environment to use for evaluation. If not set, the
		// global environment is used.
		env value.Environment
		// filename is the name of the file being read.
		filename string
	}
)

// WithEnv sets the environment to use for evaluation.
func WithEnv(env value.Environment) ReadEvalOption {
	return func(o *readEvalOptions) {
		o.env = env
	}
}

// WithFilename sets the filename to use for evaluation.
func WithFilename(filename string) ReadEvalOption {
	return func(o *readEvalOptions) {
		o.filename = filename
	}
}

// ReadEval reads and evaluates a string that may contain one or more
// forms in the global environment.
func ReadEval(code string, options ...ReadEvalOption) interface{} {
	var opts readEvalOptions
	for _, opt := range options {
		opt(&opts)
	}
	env := opts.env
	if env == nil {
		env = value.GlobalEnv
	}
	readerOpts := []reader.Option{
		reader.WithGetCurrentNS(func() string {
			return env.CurrentNamespace().Name().String()
		}),
	}
	if opts.filename != "" {
		readerOpts = append(readerOpts, reader.WithFilename(opts.filename))
	}

	r := reader.New(strings.NewReader(string(code)), readerOpts...)

	var lastValue interface{}
	for {
		expr, err := r.ReadOne()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			panic(fmt.Sprintf("error reading %v: %v", opts.filename, err))
		}
		lastValue, err = env.Eval(expr)
		if err != nil {
			panic(fmt.Sprintf("error evaluating %v: %v", opts.filename, err))
		}
	}
	return lastValue
}
