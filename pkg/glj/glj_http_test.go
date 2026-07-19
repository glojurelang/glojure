//go:build !glj_aot_runtime

package glj

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestCoreSlurpHTTPURL(t *testing.T) {
	originalClient := http.DefaultClient
	http.DefaultClient = &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("network io")),
			Request:    request,
		}, nil
	})}
	defer func() { http.DefaultClient = originalClient }()

	if got := Var("clojure.core", "slurp").Invoke("https://example.com/data"); got != "network io" {
		t.Fatalf("slurp returned %v, want %q", got, "network io")
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}
