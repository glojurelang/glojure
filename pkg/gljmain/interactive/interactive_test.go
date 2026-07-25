package interactive

import (
	"bytes"
	"strings"
	"testing"
)

func TestColorForCapturedOutput(t *testing.T) {
	var out bytes.Buffer
	if err := color(strings.NewReader("(println 42)"), &out); err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(out.Bytes(), []byte("\x1b[")) {
		t.Fatalf("color() output has no ANSI escapes: %q", out.String())
	}
	if !strings.Contains(out.String(), "\x1b[38;5;69mprintln\x1b[0m") {
		t.Fatalf("color() did not color a core symbol: %q", out.String())
	}
}

func TestParseServerArg(t *testing.T) {
	tests := []struct {
		name     string
		arg      string
		prefix   string
		host     string
		port     int
		portFile string
	}{
		{name: "default", arg: "--nrepl", prefix: "--nrepl", host: "localhost"},
		{name: "port", arg: "--nrepl=7888", prefix: "--nrepl", host: "localhost", port: 7888},
		{name: "host and port", arg: "--nrepl=0.0.0.0:7888", prefix: "--nrepl", host: "0.0.0.0", port: 7888},
		{name: "IP host", arg: "--srepl=127.0.0.1", prefix: "--srepl", host: "127.0.0.1"},
		{name: "port file", arg: "--srepl=.socket-repl-port", prefix: "--srepl", host: "localhost", portFile: ".socket-repl-port"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host, port, portFile := parseServerArg(tt.arg, tt.prefix)
			if host != tt.host || port != tt.port || portFile != tt.portFile {
				t.Fatalf("parseServerArg(%q, %q) = (%q, %d, %q), want (%q, %d, %q)",
					tt.arg, tt.prefix, host, port, portFile, tt.host, tt.port, tt.portFile)
			}
		})
	}
}
