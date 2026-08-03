package lang

import (
	"fmt"
	"io"
	"reflect"
	"sync"
)

// Class is a JVM-style Class object for host classes registered through
// pkgmap (java.lang.Math, java.util.UUID, ...). It embeds the underlying
// reflect.Type so it still passes `(instance? Class x)` (which compiles
// to `HasType(reflect.Type, x)`); the embedded field promotes all
// reflect.Type methods. Name() and String() are overridden to return
// the fully-qualified Java name so `(ns-imports *ns*)` and other forms
// print as `java.lang.Math` rather than the underlying Go type name.
type Class struct {
	reflect.Type
	JavaName      string
	acceptedTypes []reflect.Type
}

// ReadablePrinter is implemented by hosted JVM compatibility values whose
// Clojure readable representation cannot be inferred from their Go type.
type ReadablePrinter interface {
	PrintReadable(io.Writer)
}

var hostConstructors sync.Map
var hostTypeConstructors sync.Map
var readablePrinterMethods sync.Map

// RegisterReadablePrinter installs a Clojure-readable print-method for a
// hosted type that cannot implement ReadablePrinter itself.
func RegisterReadablePrinter(t reflect.Type, printer func(any, io.Writer)) {
	readablePrinterMethods.Store(t, FnFunc(func(args ...any) any {
		printer(args[0], args[1].(io.Writer))
		return nil
	}))
}

// NewClass wraps t with the given fully-qualified Java name. The Java
// name is what shows up in print-method output and (.getName c).
func NewClass(t reflect.Type, javaName string) *Class {
	return NewClassWithTypes(javaName, t)
}

// NewClassWithTypes creates a JVM-style class represented by one or more Go
// types. The first type remains the primary type used for host construction
// and static interop; the remaining types are accepted by dynamic instance?.
func NewClassWithTypes(javaName string, types ...reflect.Type) *Class {
	if len(types) == 0 {
		return &Class{JavaName: javaName}
	}
	return &Class{
		Type:          types[0],
		JavaName:      javaName,
		acceptedTypes: append([]reflect.Type(nil), types...),
	}
}

func (c *Class) accepts(v any) bool {
	if c == nil || v == nil {
		return false
	}
	vType := reflect.TypeOf(v)
	for _, accepted := range c.acceptedTypes {
		if accepted != nil && (vType == accepted || vType.AssignableTo(accepted)) {
			return true
		}
	}
	return false
}

// RegisterHostConstructor installs the implementation used by `(Class. ...)`
// for a JVM compatibility class. Registrations happen during package init.
func RegisterHostConstructor(javaName string, constructor IFn) {
	hostConstructors.Store(javaName, constructor)
}

// RegisterHostTypeConstructor preserves constructor dispatch when analysis
// has reduced a compatibility Class wrapper to its primary reflect.Type.
func RegisterHostTypeConstructor(t reflect.Type, constructor IFn) {
	hostTypeConstructors.Store(t, constructor)
}

// NewHostInstance constructs a registered JVM compatibility class, falling
// back to Go's zero-value allocation for ordinary reflect.Types.
func NewHostInstance(class any, args ...any) any {
	if c, ok := class.(*Class); ok {
		if constructor, found := hostConstructors.Load(c.JavaName); found {
			return Apply(constructor, args)
		}
		class = c.Type
	}
	t, ok := class.(reflect.Type)
	if !ok {
		panic(fmt.Sprintf("new value must be a reflect.Type, got %T", class))
	}
	if constructor, found := hostTypeConstructors.Load(t); found {
		return Apply(constructor.(IFn), args)
	}
	if len(args) != 0 {
		panic(fmt.Sprintf("new %s with args unsupported", t))
	}
	return reflect.New(t).Interface()
}

// Name shadows the embedded reflect.Type.Name() so `(.getName c)` (which
// rewrite-core turns into `.Name`) returns the JVM-canonical name.
func (c *Class) Name() string { return c.JavaName }

// String shadows the embedded reflect.Type.String() so `(str c)` and any
// fmt.Stringer caller renders the FQ Java name.
func (c *Class) String() string { return c.JavaName }

// Types returns every hosted Go type represented by this JVM compatibility
// class. Protocol extension uses this for marker classes such as Named, whose
// JVM surface covers both Keyword and Symbol values.
func (c *Class) Types() []reflect.Type {
	if c == nil {
		return nil
	}
	return append([]reflect.Type(nil), c.acceptedTypes...)
}

// classPrintMethod is the print-method body for *Class values: write the
// JavaName verbatim. Installed into the print-method MultiFn at
// construction time (see registerWellKnownMethods in multifn.go).
var classPrintMethod = FnFunc(func(args ...any) any {
	c, _ := args[0].(*Class)
	w, _ := args[1].(io.Writer)
	if c == nil || w == nil {
		return nil
	}
	io.WriteString(w, c.JavaName)
	return nil
})

var readablePrintMethod = FnFunc(func(args ...any) any {
	value, _ := args[0].(ReadablePrinter)
	writer, _ := args[1].(io.Writer)
	if value == nil || writer == nil {
		return nil
	}
	value.PrintReadable(writer)
	return nil
})
