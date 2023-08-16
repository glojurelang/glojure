package lang

import (
	"fmt"
	"strconv"
	"strings"
)

type (
	TimeoutError struct {
		msg string
	}

	// Stacker is an interface for retrieving stack traces.
	Stacker interface {
		Stack() []StackFrame
	}

	// Error is a value that represents an error.
	Error struct {
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

// NewTimeoutError creates a new timeout error.
func NewTimeoutError(msg string) error {
	return &TimeoutError{msg: msg}
}

// Error returns the error message.
func (e *TimeoutError) Error() string {
	return e.msg
}

// NewError creates a new error value.
func NewError(frame StackFrame, err error) *Error {
	return &Error{
		err:   err,
		stack: []StackFrame{frame},
	}
}

// Error returns the error message.
func (e *Error) Error() string {
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
func (e *Error) Stack() []StackFrame {
	return e.stack
}

// AddStack adds a new stack trace entry.
func (e *Error) AddStack(frame StackFrame) error {
	e.stack = append(e.stack, frame)
	return e
}
