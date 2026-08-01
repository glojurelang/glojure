package main

import (
	"flag"
	"strings"

	"github.com/glojurelang/glojure/internal/genpkg"
)

// defaultPackages contains the documented standard-library interop surface
// plus Glojure's own packages, which the runtime exposes through the same map.
// Custom package maps can select a different package set with -packages.
var defaultPackages = []string{
	"bytes",
	"context",
	"errors",
	"flag",
	"fmt",
	"io",
	"io/fs",
	"io/ioutil",
	"math",
	"math/big",
	"math/rand",
	"net/http",
	"os",
	"os/exec",
	"os/signal",
	"path/filepath",
	"reflect",
	"regexp",
	"runtime/debug",
	"sort",
	"strconv",
	"strings",
	"sync",
	"sync/atomic",
	"time",
	"unicode",

	"github.com/glojurelang/glojure/pkg/lang",
	"github.com/glojurelang/glojure/pkg/nrepl",
	"github.com/glojurelang/glojure/pkg/httpserver",
	"github.com/glojurelang/glojure/pkg/podclient",
	"github.com/glojurelang/glojure/pkg/repl",
	"github.com/glojurelang/glojure/pkg/runtime",
	"github.com/glojurelang/glojure/pkg/srepl",
}

var packagesFlag = flag.String(
	"packages",
	"",
	"comma separated list of packages to import",
)

func main() {
	flag.Parse()

	packages := defaultPackages
	if *packagesFlag != "" {
		packages = strings.Split(*packagesFlag, ",")
	}

	genpkg.GenPkgs(packages)
}
