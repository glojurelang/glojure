// Package stacktrace supplies the java.lang.StackTraceElement value used by
// portable exception-formatting libraries.
package stacktrace

import (
	"reflect"

	"github.com/glojurelang/glojure/pkg/lang"
	"github.com/glojurelang/glojure/pkg/pkgmap"
)

type StackTraceElement struct {
	className  string
	methodName string
	fileName   any
	lineNumber int32
}

func New(className, methodName string, fileName any, lineNumber any) *StackTraceElement {
	return &StackTraceElement{
		className: className, methodName: methodName, fileName: fileName,
		lineNumber: int32(lang.MustAsInt(lineNumber)),
	}
}

func (e *StackTraceElement) GetClassName() string  { return e.className }
func (e *StackTraceElement) GetMethodName() string { return e.methodName }
func (e *StackTraceElement) GetFileName() any      { return e.fileName }
func (e *StackTraceElement) GetLineNumber() int32  { return e.lineNumber }

// Link gives embedders an explicit package-retention reference.
func Link() {}

func init() {
	const javaName = "java.lang.StackTraceElement"
	pkgmap.SetHostClassPackage("StackTraceElement", "java.lang")
	pkgmap.SetHostClass("StackTraceElement",
		lang.NewClass(reflect.TypeOf((*StackTraceElement)(nil)), javaName))
	lang.RegisterHostConstructor(javaName, lang.FnFunc(func(args ...any) any {
		return New(args[0].(string), args[1].(string), args[2], args[3])
	}))
}
