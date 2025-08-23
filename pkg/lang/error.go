package lang

import (
	"fmt"
	"strconv"
	"strings"
)

type (
	Error struct {
		msg string
	}

	TimeoutError struct {
		msg string
	}

	IndexOutOfBoundsError struct{}

	IllegalArgumentError struct {
		msg string
	}

	IllegalStateError struct {
		msg string
	}

	ArithmeticError struct {
		msg string
	}

	CompilerError struct {
		file string
		line int
		col  int
		err  error
	}

	// Stacker is an interface for retrieving stack traces.
	Stacker interface {
		Stack() []StackFrame
	}

	// EvalError is a value that represents an evaluation error.
	EvalError struct {
		err   error
		stack []StackFrame
	}

	StackFrame struct {
		FunctionName string
		Filename     string
		Line         int
		Column       int
	}
)

////////////////////////////////////////////////////////////////////////////////

// NewError creates a new error value.
func NewError(msg string) error {
	return &Error{msg: msg}
}

// Error returns the error message.
func (e *Error) Error() string {
	return e.msg
}

////////////////////////////////////////////////////////////////////////////////

// NewTimeoutError creates a new timeout error.
func NewTimeoutError(msg string) error {
	return &TimeoutError{msg: msg}
}

// Error returns the error message.
func (e *TimeoutError) Error() string {
	return e.msg
}

func (e *TimeoutError) Is(other error) bool {
	_, ok := other.(*TimeoutError)
	return ok
}

////////////////////////////////////////////////////////////////////////////////

func NewIndexOutOfBoundsError() error {
	return &IndexOutOfBoundsError{}
}

func (e *IndexOutOfBoundsError) Error() string {
	return "index out of bounds"
}

func (e *IndexOutOfBoundsError) Is(other error) bool {
	_, ok := other.(*IndexOutOfBoundsError)
	return ok
}

////////////////////////////////////////////////////////////////////////////////

func NewIllegalArgumentError(msg string) error {
	return &IllegalArgumentError{msg: msg}
}

func (e *IllegalArgumentError) Error() string {
	return e.msg
}

func (e *IllegalArgumentError) Is(other error) bool {
	_, ok := other.(*IllegalArgumentError)
	return ok
}

////////////////////////////////////////////////////////////////////////////////

func NewArithmeticError(msg string) error {
	return &ArithmeticError{msg: msg}
}

func (e *ArithmeticError) Error() string {
	return e.msg
}

func (e *ArithmeticError) Is(other error) bool {
	_, ok := other.(*ArithmeticError)
	return ok
}

////////////////////////////////////////////////////////////////////////////////

func NewIllegalStateError(msg string) error {
	return &IllegalStateError{msg: msg}
}

func (e *IllegalStateError) Error() string {
	return e.msg
}

func (e *IllegalStateError) Is(other error) bool {
	_, ok := other.(*IllegalStateError)
	return ok
}

////////////////////////////////////////////////////////////////////////////////

func NewCompilerError(file string, line, col int, err error) error {
	return &CompilerError{
		file: file,
		line: line,
		col:  col,
		err:  err,
	}
}

func (e *CompilerError) Error() string {
	return fmt.Sprintf("compiler error at %s:%d:%d: %v", e.file, e.line, e.col, e.err)
}

////////////////////////////////////////////////////////////////////////////////
// TODO: Revisit

// NewEvalError creates a new error value.
func NewEvalError(frame StackFrame, err error) *EvalError {
	return &EvalError{
		err:   err,
		stack: []StackFrame{frame},
	}
}

// Error returns the error message.
func (e *EvalError) Error() string {
	var builder strings.Builder
	builder.WriteString(e.err.Error())
	builder.WriteString("\nStack trace (most recent call first):\n")
	for _, frame := range e.stack {
		builder.WriteString(frame.FunctionName)
		builder.WriteString("\n\t")
		filename := frame.Filename
		line := strconv.Itoa(frame.Line)
		column := strconv.Itoa(frame.Column)
		if filename == "" {
			filename = "<unknown>"
			line = "?"
			column = "?"
		}
		builder.WriteString(fmt.Sprintf("%s:%s:%s", filename, line, column))
		builder.WriteRune('\n')
	}
	return builder.String()
}

// Stack returns the stack trace.
func (e *EvalError) Stack() []StackFrame {
	return e.stack
}

// AddStack adds a new stack trace entry.
func (e *EvalError) AddStack(frame StackFrame) error {
	e.stack = append(e.stack, frame)
	return e
}

// Unwrap returns the underlying error.
func (e *EvalError) Unwrap() error {
	return e.err
}
