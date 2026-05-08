package nrepl

import (
	"bytes"
	"testing"
)

func TestBencodeEncodeString(t *testing.T) {
	got, err := BencodeEncode("hello")
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "5:hello" {
		t.Errorf("got %q, want %q", got, "5:hello")
	}
}

func TestBencodeEncodeEmptyString(t *testing.T) {
	got, err := BencodeEncode("")
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "0:" {
		t.Errorf("got %q, want %q", got, "0:")
	}
}

func TestBencodeEncodeInt(t *testing.T) {
	got, err := BencodeEncode(42)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "i42e" {
		t.Errorf("got %q, want %q", got, "i42e")
	}
}

func TestBencodeEncodeNegativeInt(t *testing.T) {
	got, err := BencodeEncode(-1)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "i-1e" {
		t.Errorf("got %q, want %q", got, "i-1e")
	}
}

func TestBencodeEncodeList(t *testing.T) {
	got, err := BencodeEncode([]interface{}{"a", "b"})
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "l1:a1:be" {
		t.Errorf("got %q, want %q", got, "l1:a1:be")
	}
}

func TestBencodeEncodeDict(t *testing.T) {
	got, err := BencodeEncode(map[string]interface{}{
		"op": "clone",
		"id": "1",
	})
	if err != nil {
		t.Fatal(err)
	}
	// keys sorted: id, op
	if string(got) != "d2:id1:12:op5:clonee" {
		t.Errorf("got %q, want %q", got, "d2:id1:12:op5:clonee")
	}
}

func TestBencodeDecodeString(t *testing.T) {
	val, err := BencodeDecode(bytes.NewReader([]byte("5:hello")))
	if err != nil {
		t.Fatal(err)
	}
	if val != "hello" {
		t.Errorf("got %q, want %q", val, "hello")
	}
}

func TestBencodeDecodeInt(t *testing.T) {
	val, err := BencodeDecode(bytes.NewReader([]byte("i42e")))
	if err != nil {
		t.Fatal(err)
	}
	if val != int64(42) {
		t.Errorf("got %v, want %v", val, 42)
	}
}

func TestBencodeDecodeList(t *testing.T) {
	val, err := BencodeDecode(bytes.NewReader([]byte("l1:a1:be")))
	if err != nil {
		t.Fatal(err)
	}
	list := val.([]interface{})
	if len(list) != 2 || list[0] != "a" || list[1] != "b" {
		t.Errorf("got %v, want [a b]", list)
	}
}

func TestBencodeDecodeDict(t *testing.T) {
	val, err := BencodeDecode(bytes.NewReader([]byte("d2:id1:12:op5:clonee")))
	if err != nil {
		t.Fatal(err)
	}
	dict := val.(map[string]interface{})
	if dict["id"] != "1" || dict["op"] != "clone" {
		t.Errorf("got %v", dict)
	}
}

func TestBencodeRoundTrip(t *testing.T) {
	// Simulate a real nREPL eval response
	msg := map[string]interface{}{
		"id":      "42",
		"session": "abc-123",
		"value":   "3",
		"ns":      "user",
		"status":  []interface{}{"done"},
	}
	encoded, err := BencodeEncode(msg)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := BencodeDecode(bytes.NewReader(encoded))
	if err != nil {
		t.Fatal(err)
	}
	dict := decoded.(map[string]interface{})
	if dict["id"] != "42" {
		t.Errorf("id: got %v, want 42", dict["id"])
	}
	if dict["value"] != "3" {
		t.Errorf("value: got %v, want 3", dict["value"])
	}
	status := dict["status"].([]interface{})
	if len(status) != 1 || status[0] != "done" {
		t.Errorf("status: got %v, want [done]", status)
	}
}

func TestBencodeDecodeEmptyList(t *testing.T) {
	val, err := BencodeDecode(bytes.NewReader([]byte("le")))
	if err != nil {
		t.Fatal(err)
	}
	list := val.([]interface{})
	if len(list) != 0 {
		t.Errorf("got %v, want empty list", list)
	}
}

func TestBencodeDecodeEmptyDict(t *testing.T) {
	val, err := BencodeDecode(bytes.NewReader([]byte("de")))
	if err != nil {
		t.Fatal(err)
	}
	dict := val.(map[string]interface{})
	if len(dict) != 0 {
		t.Errorf("got %v, want empty dict", dict)
	}
}
