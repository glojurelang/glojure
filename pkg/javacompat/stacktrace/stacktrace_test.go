package stacktrace

import "testing"

func TestStackTraceElementAccessors(t *testing.T) {
	element := New("foo.bar", "invoke", "bar.clj", int64(456))
	if element.GetClassName() != "foo.bar" ||
		element.GetMethodName() != "invoke" ||
		element.GetFileName() != "bar.clj" ||
		element.GetLineNumber() != 456 {
		t.Fatalf("unexpected element: %#v", element)
	}
}
