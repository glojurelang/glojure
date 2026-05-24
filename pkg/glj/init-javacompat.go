package glj

// Register JVM-faithful java.lang.* shims into pkgmap so Class/method
// forms (Math/abs, Math/PI, etc.) resolve at the REPL the same way they
// do under AOT compilation.
import (
	_ "github.com/gloathub/glojure/pkg/javacompat/boolean"
	_ "github.com/gloathub/glojure/pkg/javacompat/character"
	_ "github.com/gloathub/glojure/pkg/javacompat/double"
	_ "github.com/gloathub/glojure/pkg/javacompat/integer"
	_ "github.com/gloathub/glojure/pkg/javacompat/long"
	_ "github.com/gloathub/glojure/pkg/javacompat/math"
	_ "github.com/gloathub/glojure/pkg/javacompat/string"
	_ "github.com/gloathub/glojure/pkg/javacompat/system"
)
