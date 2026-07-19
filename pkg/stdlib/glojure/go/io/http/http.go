// Package http adds HTTP URL support to glojure.go.io.
package http

import (
	"fmt"
	"io"
	nethttp "net/http"
	"net/url"

	"github.com/glojurelang/glojure/pkg/runtime"
)

func init() {
	runtime.RegisterURLOpener(openURL)
}

func openURL(u *url.URL) (io.ReadCloser, error) {
	request, err := nethttp.NewRequest(nethttp.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}

	response, err := nethttp.DefaultClient.Do(request)
	if err != nil {
		return nil, err
	}
	if response.StatusCode != nethttp.StatusOK {
		_ = response.Body.Close()
		return nil, fmt.Errorf("http error: %d", response.StatusCode)
	}
	return response.Body, nil
}
