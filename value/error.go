package value

import (
	"fmt"
	"strconv"
	"strings"
)

// Stacker is an interface for retrieving stack traces.
type Stacker interface {
	Stack() []StackFrame
}

// Error is a value that represents an error.
type Error struct {
	err   error
	stack []StackFrame
}

type StackFrame struct {
	FunctionName string
	Pos          Pos
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
		pos := frame.Pos
		filename := pos.Filename
		line := strconv.Itoa(pos.Line)
		column := strconv.Itoa(pos.Column)
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
