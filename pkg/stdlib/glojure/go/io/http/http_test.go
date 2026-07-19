package http

import (
	"io"
	nethttp "net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/glojurelang/glojure/pkg/runtime"
)

func TestRegisteredURLOpener(t *testing.T) {
	replaceDefaultClient(t, nethttp.StatusOK, "http io")

	u, err := url.Parse("https://example.com/data")
	if err != nil {
		t.Fatal(err)
	}
	reader, err := runtime.OpenURL(u)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()

	got, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "http io" {
		t.Fatalf("response body = %q, want %q", got, "http io")
	}
}

func TestRegisteredURLOpenerRejectsNonOKStatus(t *testing.T) {
	replaceDefaultClient(t, nethttp.StatusNotFound, "")

	u, err := url.Parse("https://example.com/missing")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := runtime.OpenURL(u); err == nil {
		t.Fatal("OpenURL succeeded for a non-OK HTTP status")
	}
}

func replaceDefaultClient(t *testing.T, status int, body string) {
	t.Helper()

	originalClient := nethttp.DefaultClient
	nethttp.DefaultClient = &nethttp.Client{Transport: roundTripFunc(func(request *nethttp.Request) (*nethttp.Response, error) {
		return &nethttp.Response{
			StatusCode: status,
			Body:       io.NopCloser(strings.NewReader(body)),
			Request:    request,
		}, nil
	})}
	t.Cleanup(func() { nethttp.DefaultClient = originalClient })
}

type roundTripFunc func(*nethttp.Request) (*nethttp.Response, error)

func (fn roundTripFunc) RoundTrip(request *nethttp.Request) (*nethttp.Response, error) {
	return fn(request)
}
