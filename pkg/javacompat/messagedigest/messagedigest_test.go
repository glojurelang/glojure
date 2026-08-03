package messagedigest

import (
	"encoding/hex"
	"testing"

	"github.com/glojurelang/glojure/pkg/lang"
)

func TestMD5Digest(t *testing.T) {
	digest := GetInstance("MD5").(*MessageDigest)
	length, _ := digest.ResolveFieldOrMethod("getDigestLength")
	if got := lang.Apply(length, nil); got != int64(16) {
		t.Fatalf("digest length = %v, want 16", got)
	}
	reset, _ := digest.ResolveFieldOrMethod("reset")
	lang.Apply(reset, nil)
	update, _ := digest.ResolveFieldOrMethod("update")
	lang.Apply1(update, []byte("hello"))
	digestFn, _ := digest.ResolveFieldOrMethod("digest")
	signed := lang.Apply(digestFn, nil).([]int8)
	bytes := make([]byte, len(signed))
	for index, value := range signed {
		bytes[index] = byte(value)
	}
	got := hex.EncodeToString(bytes)
	if want := "5d41402abc4b2a76b9719d911017c592"; got != want {
		t.Fatalf("digest = %s, want %s", got, want)
	}
}

func TestSecurityProvidersEnumerateDigestAlgorithms(t *testing.T) {
	providers := GetProviders()
	keys := providers[0].(*Provider).Keys()
	var values []any
	for keys.HasMoreElements() {
		values = append(values, keys.NextElement())
	}
	if len(values) != 5 || values[2] != "MessageDigest.SHA-256" {
		t.Fatalf("provider keys = %v", values)
	}
}

func TestEnumerationSeqCreate(t *testing.T) {
	enumeration := (&Provider{algorithms: []string{"MD5", "SHA-1"}}).Keys()
	sequence := lang.Seq(EnumerationSeqCreate(enumeration))
	if got := sequence.First(); got != "MessageDigest.MD5" {
		t.Fatalf("first = %v", got)
	}
	if got := sequence.Next().First(); got != "MessageDigest.SHA-1" {
		t.Fatalf("second = %v", got)
	}
	if got := sequence.Next().Next(); got != nil {
		t.Fatalf("tail = %v", got)
	}
}
