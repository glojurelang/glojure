package srepl

import (
	"bufio"
	"fmt"
	"net"
	"strings"

	"github.com/gloathub/glojure/pkg/lang"
	"github.com/gloathub/glojure/pkg/reader"
)

func (s *Server) handleConnection(conn net.Conn) {
	defer conn.Close()

	ns := "user"
	var buf strings.Builder
	scanner := bufio.NewScanner(conn)

	fmt.Fprintf(conn, "%s=> ", ns)

	for scanner.Scan() {
		line := scanner.Text()

		if buf.Len() > 0 {
			buf.WriteRune('\n')
		}
		buf.WriteString(line)

		input := buf.String()
		if strings.TrimSpace(input) == "" {
			buf.Reset()
			fmt.Fprintf(conn, "%s=> ", ns)
			continue
		}

		if !isBalanced(input) {
			fmt.Fprintf(conn, "%s.. ", strings.Repeat(" ", len(ns)-1))
			continue
		}

		ns = evalAndPrint(conn, input, ns)
		buf.Reset()
		fmt.Fprintf(conn, "%s=> ", ns)
	}
}

// evalAndPrint evaluates code, prints results/errors to conn, and
// returns the (possibly updated) namespace name.
func evalAndPrint(conn net.Conn, code string, nsName string) string {
	ns := lang.FindNamespace(lang.NewSymbol(nsName))
	if ns == nil {
		ns = lang.FindNamespace(lang.NewSymbol("user"))
	}
	if ns == nil {
		ns = lang.FindOrCreateNamespace(lang.NewSymbol(nsName))
	}

	bindings := lang.NewMap(
		lang.VarCurrentNS, ns,
		lang.VarOut, conn,
		lang.VarWarnOnReflection, lang.VarWarnOnReflection.Deref(),
		lang.VarUncheckedMath, lang.VarUncheckedMath.Deref(),
		lang.VarDataReaders, lang.VarDataReaders.Deref(),
	)

	var lastValue string
	var evalErr error

	func() {
		lang.PushThreadBindings(bindings)
		defer lang.PopThreadBindings()
		defer func() {
			if r := recover(); r != nil {
				evalErr = fmt.Errorf("%v", r)
			}
		}()

		env := lang.GlobalEnv
		rdr := reader.New(
			strings.NewReader(code),
			reader.WithFilename("srepl"),
			reader.WithGetCurrentNS(func() *lang.Namespace {
				return env.CurrentNamespace()
			}),
		)
		vals, err := rdr.ReadAll()
		if err != nil {
			evalErr = err
			return
		}
		for _, val := range vals {
			result, err := env.Eval(val)
			if err != nil {
				evalErr = err
				return
			}
			lastValue = lang.PrintString(result)
		}
		nsName = env.CurrentNamespace().Name().String()
	}()

	if evalErr != nil {
		fmt.Fprintf(conn, "Error: %v\n", evalErr)
	} else {
		fmt.Fprintf(conn, "%s\n", lastValue)
	}

	return nsName
}

// isBalanced returns true if parentheses, brackets, and braces are balanced.
func isBalanced(input string) bool {
	depth := 0
	inString := false
	escape := false
	for _, r := range input {
		if escape {
			escape = false
			continue
		}
		if r == '\\' && inString {
			escape = true
			continue
		}
		if r == '"' {
			inString = !inString
			continue
		}
		if inString {
			continue
		}
		switch r {
		case '(', '[', '{':
			depth++
		case ')', ']', '}':
			depth--
		}
	}
	return depth <= 0 && !inString
}
