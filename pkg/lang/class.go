package lang

import (
	"io"
	"reflect"
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
	JavaName string
}

// NewClass wraps t with the given fully-qualified Java name. The Java
// name is what shows up in print-method output and (.getName c).
func NewClass(t reflect.Type, javaName string) *Class {
	return &Class{Type: t, JavaName: javaName}
}

// Name shadows the embedded reflect.Type.Name() so `(.getName c)` (which
// rewrite-core turns into `.Name`) returns the JVM-canonical name.
func (c *Class) Name() string { return c.JavaName }

// String shadows the embedded reflect.Type.String() so `(str c)` and any
// fmt.Stringer caller renders the FQ Java name.
func (c *Class) String() string { return c.JavaName }

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
