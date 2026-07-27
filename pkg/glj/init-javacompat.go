package glj

// Register JVM-faithful java.lang.* shims into pkgmap so Class/method
// forms (Math/abs, Math/PI, etc.) resolve at the REPL the same way they
// do under AOT compilation.
import (
	_ "github.com/glojurelang/glojure/pkg/javacompat/boolean"
	_ "github.com/glojurelang/glojure/pkg/javacompat/character"
	_ "github.com/glojurelang/glojure/pkg/javacompat/double"
	_ "github.com/glojurelang/glojure/pkg/javacompat/instant"
	_ "github.com/glojurelang/glojure/pkg/javacompat/integer"
	_ "github.com/glojurelang/glojure/pkg/javacompat/long"
	_ "github.com/glojurelang/glojure/pkg/javacompat/mapentry"
	_ "github.com/glojurelang/glojure/pkg/javacompat/math"
	_ "github.com/glojurelang/glojure/pkg/javacompat/regex"
	_ "github.com/glojurelang/glojure/pkg/javacompat/string"
	_ "github.com/glojurelang/glojure/pkg/javacompat/system"
	_ "github.com/glojurelang/glojure/pkg/javacompat/thread"
	_ "github.com/glojurelang/glojure/pkg/javacompat/uuid"
)
