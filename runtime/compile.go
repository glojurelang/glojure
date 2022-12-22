package runtime

import (
	"github.com/glojurelang/glojure/value"
)

type (
	gljContext int
)

const (
	statementCtx gljContext = iota
	expressionCtx
	returnCtx
	evalCtx
)

var (
	symbolDo = value.NewSymbol("do")
)

// func Compile(env *environment, rdr *reader.Reader) error {
// 	for {
// 		form, err := rdr.ReadOne()
// 		if errors.Is(err, io.EOF) {
// 			break
// 		}
// 		if err != nil {
// 			return err
// 		}

// 		if err := compileForm(form); err != nil {
// 			return err
// 		}
// 	}
// }

// func compileForm(env *environment, form interface{}) error {
// 	form = macroexpand(env, form)
// 	if fseq, ok := form.(value.ISeq); ok && value.Equal(value.First(fseq), symbolDo) {
// 		for fseq = value.Next(fseq); fseq != nil; fseq = value.Next(fseq) {
// 			if err := compileForm(env, value.First(fseq)); err != nil {
// 				return err
// 			}
// 		}
// 		return nil
// 	}

// 	_, err := analyze(evalCtx, env, form)
// 	if err != nil {
// 		return err
// 	}
// 	return nil
// }

// func analyze(ctx context, env *environment, form interface{}) (interface{}, error) {
// 	switch form := form.(type) {
// 	case value.ISymbol:
// 		return analyzeSymbol(ctx, env, form)
// 	case value.ISeq:
// 		return analyzeSeq(ctx, env, form)
// 	default:
// 		return analyzeLiteral(ctx, env, form)
// 	}
// }
