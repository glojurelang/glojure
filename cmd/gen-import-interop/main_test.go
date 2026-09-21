package main

import (
	"slices"
	"testing"
)

func TestDefaultPackagesMatchSupportedPackageMap(t *testing.T) {
	want := []string{
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
		"runtime",
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
	if !slices.Equal(defaultPackages, want) {
		t.Fatalf("default packages = %v, want %v", defaultPackages, want)
	}
}
