package lang

import (
	"errors"
	"fmt"
	"io"
	"reflect"
	"testing"
)

// Custom error types for testing
type CustomError struct {
	msg string
}

func (e *CustomError) Error() string { return e.msg }

type WrappedError struct {
	inner error
}

func (e *WrappedError) Error() string { return fmt.Sprintf("wrapped: %v", e.inner) }
func (e *WrappedError) Unwrap() error { return e.inner }

// Custom non-error types for testing
type MyStruct struct {
	Value int
}

type MyInterface interface {
	DoSomething()
}

type MyImpl struct{}

func (m MyImpl) DoSomething() {}

func TestCatchMatches(t *testing.T) {
	customErr := &CustomError{msg: "custom"}
	wrappedErr := &WrappedError{inner: customErr}
	basicErr := errors.New("basic error")
	eofErr := io.EOF

	tests := []struct {
		name   string
		r      any
		expect any
		want   bool
	}{
		// Nil expect cases
		{
			name:   "nil expect returns false",
			r:      errors.New("any error"),
			expect: nil,
			want:   false,
		},
		{
			name:   "nil expect with nil r returns false",
			r:      nil,
			expect: nil,
			want:   false,
		},

		// Basic error interface matching
		{
			name:   "basic error matches error interface",
			r:      basicErr,
			expect: errorType,
			want:   true,
		},
		{
			name:   "custom error matches error interface",
			r:      customErr,
			expect: errorType,
			want:   true,
		},
		{
			name:   "wrapped error matches error interface",
			r:      wrappedErr,
			expect: errorType,
			want:   true,
		},

		// Specific error type matching - pointer types
		{
			name:   "custom error matches its own pointer type",
			r:      customErr,
			expect: reflect.TypeOf((*CustomError)(nil)),
			want:   true,
		},
		{
			name:   "wrapped error matches its own pointer type",
			r:      wrappedErr,
			expect: reflect.TypeOf((*WrappedError)(nil)),
			want:   true,
		},
		{
			name:   "wrapped error matches inner error pointer type via Unwrap",
			r:      wrappedErr,
			expect: reflect.TypeOf((*CustomError)(nil)),
			want:   true,
		},
		{
			name:   "custom error does not match different error pointer type",
			r:      customErr,
			expect: reflect.TypeOf((*WrappedError)(nil)),
			want:   false,
		},
		{
			name:   "ExceptionInfo matches IExceptionInfo interface",
			r:      NewExceptionInfo("msg", nil),
			expect: reflect.TypeOf((*IExceptionInfo)(nil)).Elem(),
			want:   true,
		},
		{
			name:   "Wrapped ExceptionInfo matches IExceptionInfo interface",
			r:      fmt.Errorf("wrapped: %w", NewExceptionInfo("msg", nil)),
			expect: reflect.TypeOf((*IExceptionInfo)(nil)).Elem(),
			want:   true,
		},

		// Interface type matching for errors
		{
			name:   "EOF error matches error interface",
			r:      eofErr,
			expect: errorType,
			want:   true,
		},

		// Non-error type matching
		{
			name:   "string matches string type",
			r:      "hello",
			expect: reflect.TypeOf(""),
			want:   true,
		},
		{
			name:   "int matches int type",
			r:      42,
			expect: reflect.TypeOf(0),
			want:   true,
		},
		{
			name:   "struct matches its own type",
			r:      MyStruct{Value: 10},
			expect: reflect.TypeOf(MyStruct{}),
			want:   true,
		},
		{
			name:   "pointer to struct matches pointer type",
			r:      &MyStruct{Value: 10},
			expect: reflect.TypeOf((*MyStruct)(nil)),
			want:   true,
		},
		{
			name:   "struct does not match pointer to struct type",
			r:      MyStruct{Value: 10},
			expect: reflect.TypeOf((*MyStruct)(nil)),
			want:   false,
		},

		// Interface implementation matching
		{
			name:   "implementation matches interface type",
			r:      MyImpl{},
			expect: reflect.TypeOf((*MyInterface)(nil)).Elem(),
			want:   true,
		},
		{
			name:   "pointer to implementation matches interface type",
			r:      &MyImpl{},
			expect: reflect.TypeOf((*MyInterface)(nil)).Elem(),
			want:   true,
		},
		{
			name:   "non-implementation does not match interface type",
			r:      MyStruct{},
			expect: reflect.TypeOf((*MyInterface)(nil)).Elem(),
			want:   false,
		},

		// Type mismatch cases
		{
			name:   "string does not match int type",
			r:      "hello",
			expect: reflect.TypeOf(0),
			want:   false,
		},
		{
			name:   "int does not match string type",
			r:      42,
			expect: reflect.TypeOf(""),
			want:   false,
		},
		{
			name:   "error does not match non-error struct type",
			r:      errors.New("error"),
			expect: reflect.TypeOf(MyStruct{}),
			want:   false,
		},
		{
			name:   "struct does not match error interface",
			r:      MyStruct{Value: 10},
			expect: errorType,
			want:   false,
		},

		// Nil value cases
		{
			name:   "nil r with error type returns false",
			r:      nil,
			expect: errorType,
			want:   false,
		},
		{
			name:   "nil r with struct type returns false",
			r:      nil,
			expect: reflect.TypeOf(MyStruct{}),
			want:   false,
		},
		{
			name:   "typed nil error does not match error interface",
			r:      (*CustomError)(nil),
			expect: errorType,
			want:   false,
		},

		// Complex type hierarchies
		{
			name:   "interface{} type accepts any non-nil value",
			r:      "anything",
			expect: reflect.TypeOf((*any)(nil)).Elem(),
			want:   true,
		},
		{
			name:   "interface{} type accepts error",
			r:      errors.New("error"),
			expect: reflect.TypeOf((*any)(nil)).Elem(),
			want:   true,
		},
		{
			name:   "interface{} type accepts struct",
			r:      MyStruct{},
			expect: reflect.TypeOf((*any)(nil)).Elem(),
			want:   true,
		},

		// Array and slice types
		{
			name:   "slice matches slice type",
			r:      []int{1, 2, 3},
			expect: reflect.TypeOf([]int{}),
			want:   true,
		},
		{
			name:   "array matches array type",
			r:      [3]int{1, 2, 3},
			expect: reflect.TypeOf([3]int{}),
			want:   true,
		},
		{
			name:   "slice does not match array type",
			r:      []int{1, 2, 3},
			expect: reflect.TypeOf([3]int{}),
			want:   false,
		},

		// Map types
		{
			name:   "map matches map type",
			r:      map[string]int{"a": 1},
			expect: reflect.TypeOf(map[string]int{}),
			want:   true,
		},
		{
			name:   "map with different key type does not match",
			r:      map[string]int{"a": 1},
			expect: reflect.TypeOf(map[int]int{}),
			want:   false,
		},

		// Channel types
		{
			name:   "channel matches channel type",
			r:      make(chan int),
			expect: reflect.TypeOf(make(chan int)),
			want:   true,
		},
		{
			name:   "buffered channel matches unbuffered channel type",
			r:      make(chan int, 10),
			expect: reflect.TypeOf(make(chan int)),
			want:   true,
		},
		{
			name:   "send-only channel does not match receive-only channel",
			r:      make(chan<- int),
			expect: reflect.TypeOf(make(<-chan int)),
			want:   false,
		},

		// Function types
		{
			name:   "function matches function type",
			r:      func() {},
			expect: reflect.TypeOf(func() {}),
			want:   true,
		},
		{
			name:   "function with different signature does not match",
			r:      func() {},
			expect: reflect.TypeOf(func(int) {}),
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CatchMatches(tt.r, tt.expect)
			if got != tt.want {
				t.Errorf("CatchMatches(%v, %v) = %v, want %v", tt.r, tt.expect, got, tt.want)
			}
		})
	}
}

// Test to ensure CatchMatches handles panic recovery scenarios correctly
func TestCatchMatchesPanicRecovery(t *testing.T) {
	tests := []struct {
		name        string
		panicVal    any
		catchType   any
		shouldMatch bool
	}{
		{
			name:        "panic with error catches as error",
			panicVal:    errors.New("panic error"),
			catchType:   errorType,
			shouldMatch: true,
		},
		{
			name:        "panic with string catches as string",
			panicVal:    "panic string",
			catchType:   reflect.TypeOf(""),
			shouldMatch: true,
		},
		{
			name:        "panic with custom error catches as custom error",
			panicVal:    &CustomError{msg: "panic"},
			catchType:   reflect.TypeOf((*CustomError)(nil)),
			shouldMatch: true,
		},
		{
			name:        "panic with int does not catch as string",
			panicVal:    42,
			catchType:   reflect.TypeOf(""),
			shouldMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					matched := CatchMatches(r, tt.catchType)
					if matched != tt.shouldMatch {
						t.Errorf("CatchMatches(recovered %v, %v) = %v, want %v",
							r, tt.catchType, matched, tt.shouldMatch)
					}
				}
			}()
			panic(tt.panicVal)
		})
	}
}
