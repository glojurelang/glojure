//go:build !wasm

package goid

import "github.com/modern-go/gls"

func Get() int64 {
	return gls.GoID()
}
