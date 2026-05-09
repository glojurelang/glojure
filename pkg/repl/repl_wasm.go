//go:build wasm

package repl

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Start starts the REPL (WASM version).
func Start(opts ...Option) {
	o := initOptions(opts)
	printBanner(o.stdout, "")

	rl := bufio.NewReader(os.Stdin)
	var expr string

	fmt.Print(defaultPrompt(&o))
	for {
		line, err := rl.ReadString('\n')
		if err != nil {
			break
		}
		expr += line

		if !isExpressionComplete(expr, o.env) {
			fmt.Print(strings.Repeat(" ", len(defaultPrompt(&o))))
			continue
		}

		trimmed := strings.TrimSpace(expr)
		if trimmed != "" {
			if trimmed == ":repl/help" {
				printHelp(o.stdout, "vi", "cat", "", noColors)
			} else {
				handled, exit := handleReplCommand(trimmed, &o)
				if exit {
					return
				}
				if !handled {
					readEvalPrint(expr, &o, evalSafe)
				}
			}
		}

		expr = ""
		fmt.Print(defaultPrompt(&o))
	}
}


