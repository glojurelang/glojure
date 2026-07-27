package ast

import (
	"strings"
	"testing"

	"github.com/glojurelang/glojure/pkg/lang"
)

func TestFormat(t *testing.T) {
	x := lang.NewSymbol("x")
	root := &Node{
		Op: OpLet,
		Sub: &LetNode{
			Bindings: []*Node{{
				Op: OpBinding,
				Sub: &BindingNode{
					Name:  x,
					Local: lang.KWLet,
					Init: &Node{
						Op:        OpConst,
						IsLiteral: true,
						Sub: &ConstNode{
							Type:  lang.KWNumber,
							Value: int64(1),
						},
					},
				},
			}},
			Body: &Node{
				Op: OpLocal,
				Sub: &LocalNode{
					Name:  x,
					Local: lang.KWLet,
				},
			},
		},
	}

	got := Format(root)
	for _, want := range []string{
		"(let",
		":bindings [(binding",
		":name x",
		":local :let",
		":init (const :literal true",
		":type :number",
		":value 1",
		":body (local",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("Format() missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "0x") {
		t.Fatalf("Format() contains an unstable address:\n%s", got)
	}
}

func TestNodeOpStringCoversAllOps(t *testing.T) {
	for op := OpUnknown; op <= OpThrow; op++ {
		if got := op.String(); strings.HasPrefix(got, "op-") {
			t.Errorf("NodeOp(%d) is missing a stable name", op)
		}
	}
}
