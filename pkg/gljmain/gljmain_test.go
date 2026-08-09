package gljmain

import (
	"os"
	"os/exec"
	"slices"
	"strings"
	"testing"
)

func TestDashStdin(t *testing.T) {
	const helperEnv = "GLJMAIN_TEST_DASH_STDIN_HELPER"
	if os.Getenv(helperEnv) == "1" {
		Main([]string{"-", "alpha", "beta"})
		os.Exit(0)
	}

	cmd := exec.Command(os.Args[0], "-test.run=^TestDashStdin$")
	cmd.Env = append(os.Environ(), helperEnv+"=1")
	cmd.Stdin = strings.NewReader("(println 123)\n(println *command-line-args*)\n")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("glj - failed: %v\n%s", err, output)
	}
	if got, want := string(output), "123\n(alpha beta)\n"; got != want {
		t.Errorf("glj - output = %q, want %q", got, want)
	}
}

func TestUsesProjectDeps(t *testing.T) {
	tests := []struct {
		args []string
		want bool
	}{
		{nil, true},
		{[]string{"main.clj"}, true},
		{[]string{"-"}, true},
		{[]string{"-e", "(+ 1 2)"}, true},
		{[]string{"--nrepl"}, true},
		{[]string{"--nrepl=7888"}, true},
		{[]string{"--srepl"}, true},
		{[]string{"--srepl=7777"}, true},
		{[]string{"--help"}, false},
		{[]string{"--version"}, false},
		{[]string{"--color"}, false},
		{[]string{"--nrepl-connect", "localhost:7888"}, false},
		{[]string{"--unknown"}, false},
	}
	for _, test := range tests {
		if got := usesProjectDeps(test.args); got != test.want {
			t.Errorf("usesProjectDeps(%q) = %v, want %v", test.args, got, test.want)
		}
	}
}

func TestSplitDepsOption(t *testing.T) {
	tests := []struct {
		args      []string
		wantEDN   string
		wantArgs  []string
		wantError bool
	}{
		{[]string{"main.clj"}, "", []string{"main.clj"}, false},
		{[]string{"-Sdeps", "{:deps {}}", "main.clj"}, "{:deps {}}", []string{"main.clj"}, false},
		{[]string{"-Sdeps", "{}"}, "{}", nil, false},
		{[]string{"-Sdeps"}, "", nil, true},
	}
	for _, test := range tests {
		gotEDN, gotArgs, err := splitDepsOption(test.args)
		if gotEDN != test.wantEDN || !slices.Equal(gotArgs, test.wantArgs) || (err != nil) != test.wantError {
			t.Errorf("splitDepsOption(%q) = (%q, %q, %v), want (%q, %q, error=%v)",
				test.args, gotEDN, gotArgs, err, test.wantEDN, test.wantArgs, test.wantError)
		}
	}
}
