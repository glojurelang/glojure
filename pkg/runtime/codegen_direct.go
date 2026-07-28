//go:build !glj_aot_runtime

package runtime

import (
	"fmt"
	"math/bits"
	"sort"
	"strings"

	"github.com/glojurelang/glojure/pkg/ast"
	"github.com/glojurelang/glojure/pkg/lang"
)

func sortedAOTTargets(
	targets map[*aotSpecializationTarget]struct{},
) []*aotSpecializationTarget {
	result := make([]*aotSpecializationTarget, 0, len(targets))
	for target := range targets {
		result = append(result, target)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].directFnVar < result[j].directFnVar
	})
	return result
}

// prepareAOTCallTargets allocates package-level call slots for ordinary
// functions in the namespace. With direct linking, inferred calls use these
// slots without consulting the Var. When direct linking is disabled, or a Var
// is marked ^:redef, generated callers guard the slot with the root seen by
// LoadNS and fall back to Var dispatch after a redefinition.
func (g *Generator) prepareAOTCallTargets(vars []namedVar) {
	g.prepareAOTProtocolTargets(vars)
	for _, named := range vars {
		vr := named.vr
		if !vr.IsBound() || vr.IsMacro() ||
			RT.BooleanCast(lang.Get(vr.Meta(), lang.KWDynamic)) {
			continue
		}

		value := codegenVarValue(vr)
		fn, ok := value.(*Fn)
		if !ok {
			continue
		}
		fnNode := fn.ASTNode().Sub.(*ast.FnNode)
		arityDispatch := len(fnNode.Methods) != 1 || fnNode.IsVariadic
		directArities := directAOTFnArities(fnNode)
		if !hasDirectAOTArity(directArities) {
			continue
		}

		index := len(g.aotCallTargets)
		directLinked := g.directLink &&
			!RT.BooleanCast(lang.Get(vr.Meta(), lang.KWRedef))
		rootVersionVar := ""
		if !directLinked {
			rootVersionVar = fmt.Sprintf("aotRootVersion%d", index)
		}
		target := &aotSpecializationTarget{
			vr:             vr,
			fn:             fn,
			arityDispatch:  arityDispatch,
			directLinked:   directLinked,
			directArities:  directArities,
			directFnVar:    fmt.Sprintf("aotDirectFn%d", index),
			int64FnVar:     fmt.Sprintf("aotInt64Fn%d", index),
			float64FnVar:   fmt.Sprintf("aotFloat64Fn%d", index),
			vectorFnVar:    fmt.Sprintf("aotVectorFn%d", index),
			recordFnVar:    fmt.Sprintf("aotRecordFn%d", index),
			rootVersionVar: rootVersionVar,
		}
		target.recordAnalysis = g.aotRecordPlans[vr]
		directType := "lang.ArityFn"
		if !arityDispatch {
			method := fnNode.Methods[0].Sub.(*ast.FnMethodNode)
			target.arity = method.FixedArity
			directType = fmt.Sprintf("lang.FnFunc%d", target.arity)
		}
		g.aotCallTargets[vr] = target
		fmt.Fprintf(&g.aotDeclarations, "var %s %s\n",
			target.directFnVar, directType)
		if target.rootVersionVar != "" {
			fmt.Fprintf(&g.aotDeclarations,
				"var %s *lang.VarRootVersion\n", target.rootVersionVar)
		}
		if arityDispatch {
			for _, methodNode := range fnNode.Methods {
				method := methodNode.Sub.(*ast.FnMethodNode)
				if method.IsVariadic ||
					method.FixedArity < 0 ||
					method.FixedArity >= len(target.directArityVars) {
					continue
				}
				slot := fmt.Sprintf(
					"%sArity%d",
					target.directFnVar,
					method.FixedArity,
				)
				target.directArityVars[method.FixedArity] = slot
				fmt.Fprintf(
					&g.aotDeclarations,
					"var %s lang.FnFunc%d\n",
					slot,
					method.FixedArity,
				)
			}
		}
	}
	for {
		changed := false
		for _, named := range vars {
			target := g.aotCallTargets[named.vr]
			if target == nil || target.arityDispatch ||
				target.arity > 4 || target.int64Analysis != nil {
				continue
			}
			fnNode := target.fn.ASTNode().Sub.(*ast.FnNode)
			method := fnNode.Methods[0].Sub.(*ast.FnMethodNode)
			analysis := analyzeInt64AOTFunction(
				target,
				method,
				g.aotCallTargets,
			)
			if analysis == nil {
				continue
			}
			target.int64Analysis = analysis
			changed = true
		}
		if !changed {
			break
		}
	}
	for {
		changed := false
		for _, named := range vars {
			target := g.aotCallTargets[named.vr]
			if target == nil || target.arityDispatch ||
				target.arity > 4 || !target.directLinked {
				continue
			}
			fnNode := target.fn.ASTNode().Sub.(*ast.FnNode)
			method := fnNode.Methods[0].Sub.(*ast.FnMethodNode)
			analysis := analyzeVectorAOTFunction(
				target,
				method,
				g.aotCallTargets,
			)
			if analysis == nil {
				continue
			}
			if target.vectorAnalysis != nil &&
				bits.OnesCount32(target.vectorAnalysis.paramMask) >=
					bits.OnesCount32(analysis.paramMask) {
				continue
			}
			target.vectorAnalysis = analysis
			changed = true
		}
		if !changed {
			break
		}
	}
	for _, named := range vars {
		target := g.aotCallTargets[named.vr]
		if target == nil || target.arityDispatch ||
			target.int64Analysis == nil {
			continue
		}
		fnNode := target.fn.ASTNode().Sub.(*ast.FnNode)
		method := fnNode.Methods[0].Sub.(*ast.FnMethodNode)
		target.int64Analysis.proveSafeOperations(method)
	}
	for {
		changed := false
		for _, named := range vars {
			target := g.aotCallTargets[named.vr]
			if target == nil || target.arityDispatch ||
				target.arity > 4 || target.int64Analysis != nil ||
				target.float64Analysis != nil {
				continue
			}
			fnNode := target.fn.ASTNode().Sub.(*ast.FnNode)
			method := fnNode.Methods[0].Sub.(*ast.FnMethodNode)
			analysis := analyzeFloat64AOTFunction(
				target,
				method,
				g.aotCallTargets,
			)
			if analysis == nil {
				continue
			}
			target.float64Analysis = analysis
			changed = true
		}
		if !changed {
			break
		}
	}
	for _, named := range vars {
		target := g.aotCallTargets[named.vr]
		if target == nil {
			continue
		}
		if target.int64Analysis != nil {
			params := make([]string, target.arity)
			for i := range params {
				params[i] = "int64"
			}
			fmt.Fprintf(
				&g.aotDeclarations,
				"var %s func(%s) (int64, bool)\n",
				target.int64FnVar,
				strings.Join(params, ", "),
			)
		}
		if target.float64Analysis != nil {
			params := make([]string, target.arity)
			for i := range params {
				params[i] = "float64"
			}
			fmt.Fprintf(
				&g.aotDeclarations,
				"var %s func(%s) (float64, bool)\n",
				target.float64FnVar,
				strings.Join(params, ", "),
			)
		}
		if target.vectorAnalysis != nil &&
			target.vectorAnalysis.result.transient {
			params := vectorAOTParamTypes(target.vectorAnalysis)
			fmt.Fprintf(
				&g.aotDeclarations,
				"var %s func(%s) *lang.TransientVector\n",
				target.vectorFnVar,
				strings.Join(params, ", "),
			)
		}
		if target.recordAnalysis != nil {
			params := make([]string, len(target.recordAnalysis.Signature.Params))
			for index, typ := range target.recordAnalysis.Signature.Params {
				params[index] = recordAOTTypeExpr(g, typ)
			}
			fmt.Fprintf(
				&g.aotDeclarations,
				"var %s func(%s) %s\n",
				target.recordFnVar,
				strings.Join(params, ", "),
				recordAOTTypeExpr(g, target.recordAnalysis.Signature.Result),
			)
		}
	}
	if len(g.aotCallTargets) > 0 {
		g.aotDeclarations.WriteByte('\n')
	}
}

func (g *Generator) aotProtocolInvokeTarget(
	invoke *ast.InvokeNode,
) *aotProtocolCallTarget {
	if invoke.Fn.Op != ast.OpVar || len(invoke.Args) > 5 {
		return nil
	}
	return g.aotProtocolCallTargets[invoke.Fn.Sub.(*ast.VarNode).Var]
}

func directAOTFnArities(fn *ast.FnNode) [21]bool {
	var result [21]bool
	for _, methodNode := range fn.Methods {
		method := methodNode.Sub.(*ast.FnMethodNode)
		if method.IsVariadic {
			if method.FixedArity < 0 {
				continue
			}
			for arity := method.FixedArity; arity < len(result); arity++ {
				result[arity] = true
			}
			continue
		}
		if method.FixedArity >= 0 && method.FixedArity < len(result) {
			result[method.FixedArity] = true
		}
	}
	return result
}

func hasDirectAOTArity(arities [21]bool) bool {
	for _, supported := range arities {
		if supported {
			return true
		}
	}
	return false
}

func codegenVarValue(vr *lang.Var) any {
	// Dynamic Vars are resolved against goroutine-local bindings. Reading from
	// a fresh goroutine obtains the root value used to build an AOT loader.
	value := make(chan any)
	go func() {
		value <- vr.Get()
	}()
	return <-value
}

func (g *Generator) aotInvokeTarget(
	invoke *ast.InvokeNode,
) *aotSpecializationTarget {
	if invoke.Fn.Op != ast.OpVar {
		return nil
	}
	target := g.aotCallTargets[invoke.Fn.Sub.(*ast.VarNode).Var]
	arity := len(invoke.Args)
	if target == nil || arity >= len(target.directArities) ||
		!target.directArities[arity] {
		return nil
	}
	return target
}

// aotExternalInvokeTarget prepares a statically resolved call into another
// namespace. Statically resolved calls are linked directly when enabled.
// Dynamic and ^:redef Vars always retain runtime lookup.
func (g *Generator) aotExternalInvokeTarget(
	invoke *ast.InvokeNode,
) *aotExternalCallTarget {
	if invoke.Fn.Op != ast.OpVar {
		return nil
	}
	vr := invoke.Fn.Sub.(*ast.VarNode).Var
	if vr.Namespace() == g.aotNamespace ||
		!vr.IsBound() ||
		vr.IsMacro() ||
		vr.IsDynamic() {
		return nil
	}
	arity := len(invoke.Args)
	intrinsic := g.aotExternalIntrinsic(vr, invoke)
	if arity > 5 ||
		(intrinsic == "" && !aotSupportsArity(codegenVarValue(vr), arity)) {
		return nil
	}
	key := aotExternalCallKey{vr: vr, arity: arity, intrinsic: intrinsic}
	if target := g.aotExternalCallTargets[key]; target != nil {
		return target
	}
	index := len(g.aotExternalCallTargets)
	directLinked := g.directLink &&
		!RT.BooleanCast(lang.Get(vr.Meta(), lang.KWRedef))
	target := &aotExternalCallTarget{
		vr:             vr,
		arity:          arity,
		fnVar:          fmt.Sprintf("aotExternalFn%d", index),
		intrinsic:      intrinsic,
		directLinked:   directLinked,
		defaultVar:     fmt.Sprintf("aotExternalDefault%d", index),
		rootVersionVar: fmt.Sprintf("aotExternalRootVersion%d", index),
	}
	if intrinsic == "" || !directLinked {
		g.allocVarVar(
			vr.Namespace().Name().String(),
			vr.Symbol().String(),
		)
	}
	g.aotExternalCallTargets[key] = target
	return target
}

func (g *Generator) aotExternalIntrinsic(
	vr *lang.Var,
	invoke *ast.InvokeNode,
) string {
	if vr.Namespace().Name().String() != "clojure.core" {
		return ""
	}
	name := vr.Symbol().String()
	arity := len(invoke.Args)
	switch {
	case name == "=" && arity == 2:
	case name == "assoc" && (arity == 3 || arity == 5):
	case name == "count" && arity == 1:
	case name == "dec" && arity == 1:
	case name == "cons" && arity == 2:
	case name == "conj" && arity == 2:
	case name == "empty?" && arity == 1:
	case name == "first" && arity == 1:
	case name == "get" && (arity == 2 || arity == 3):
	case name == "inc" && arity == 1:
	case name == "instance?" && arity == 2:
		if _, ok := g.staticInstanceCall(invoke, []string{"", "value"}); !ok {
			return ""
		}
	case name == "next" && arity == 1:
	case name == "nth" && (arity == 2 || arity == 3):
	case name == "peek" && arity == 1:
	case name == "pop" && arity == 1:
	case name == "seq" && arity == 1:
	case name == "subs" && (arity == 2 || arity == 3):
	default:
		return ""
	}
	return name
}

func (g *Generator) aotExternalIntrinsicCall(
	intrinsic string,
	invoke *ast.InvokeNode,
	args []string,
) string {
	switch intrinsic {
	case "=":
		leftInt := g.irHasInt64Representation(invoke.Args[0])
		rightInt := g.irHasInt64Representation(invoke.Args[1])
		switch {
		case leftInt && rightInt:
			return fmt.Sprintf("(%s == %s)", args[0], args[1])
		case leftInt:
			return fmt.Sprintf("lang.EqualsInt64(%s, %s)", args[0], args[1])
		case rightInt:
			return fmt.Sprintf("lang.EqualsInt64(%s, %s)", args[1], args[0])
		}
		return fmt.Sprintf("lang.Equals(%s, %s)", args[0], args[1])
	case "assoc":
		if len(args) == 3 {
			return fmt.Sprintf("lang.Assoc(%s, %s, %s)", args[0], args[1], args[2])
		}
		return fmt.Sprintf(
			"lang.Assoc(lang.Assoc(%s, %s, %s), %s, %s)",
			args[0], args[1], args[2], args[3], args[4],
		)
	case "count":
		return fmt.Sprintf("lang.Count(%s)", args[0])
	case "dec":
		return fmt.Sprintf("lang.Numbers.Dec(%s)", args[0])
	case "cons":
		return fmt.Sprintf("lang.NewCons(%s, %s)", args[0], args[1])
	case "conj":
		return fmt.Sprintf("lang.ConjAny(%s, %s)", args[0], args[1])
	case "empty?":
		return fmt.Sprintf("lang.IsEmpty(%s)", args[0])
	case "first":
		return fmt.Sprintf("lang.First(%s)", args[0])
	case "get":
		if len(args) == 2 {
			return fmt.Sprintf("lang.Get(%s, %s)", args[0], args[1])
		}
		return fmt.Sprintf("lang.GetDefault(%s, %s, %s)", args[0], args[1], args[2])
	case "inc":
		return fmt.Sprintf("lang.Numbers.Inc(%s)", args[0])
	case "instance?":
		call, ok := g.staticInstanceCall(invoke, args)
		if !ok {
			panic("static instance? intrinsic lost its target type")
		}
		return call
	case "next":
		return fmt.Sprintf("lang.Next(%s)", args[0])
	case "nth":
		if len(args) == 2 {
			return fmt.Sprintf(
				"runtime.RT.Nth(%s, lang.IntCast(%s))",
				args[0], args[1],
			)
		}
		return fmt.Sprintf(
			"runtime.RT.NthDefault(%s, lang.IntCast(%s), %s)",
			args[0], args[1], args[2],
		)
	case "peek":
		return fmt.Sprintf("runtime.RT.Peek(%s)", args[0])
	case "pop":
		return fmt.Sprintf("runtime.RT.Pop(%s)", args[0])
	case "seq":
		return fmt.Sprintf("lang.Seq(%s)", args[0])
	case "subs":
		if len(args) == 2 {
			return fmt.Sprintf(
				"runtime.RT.Subs(any(%s).(string), lang.IntCast(%s))",
				args[0], args[1],
			)
		}
		return fmt.Sprintf(
			"runtime.RT.SubsEnd(any(%s).(string), lang.IntCast(%s), lang.IntCast(%s))",
			args[0], args[1], args[2],
		)
	default:
		panic("unsupported AOT external intrinsic: " + intrinsic)
	}
}

func (g *Generator) generateAOTExternalAdapters() {
	var cachedArities, linkedArities [6]bool
	for _, target := range g.aotExternalCallTargets {
		if target.directLinked {
			linkedArities[target.arity] = true
		} else {
			cachedArities[target.arity] = true
		}
	}
	if hasAOTAdapterArity(linkedArities) {
		g.addImport("sync")
	}
	for arity, used := range cachedArities {
		if !used {
			continue
		}
		params := make([]string, arity)
		args := make([]string, arity)
		for i := range params {
			args[i] = fmt.Sprintf("p%d", i)
			params[i] = args[i] + " any"
		}
		paramList := strings.Join(params, ", ")
		argList := strings.Join(args, ", ")
		fmt.Fprintf(&g.aotDeclarations,
			"func aotCacheFn%d(vr *lang.Var) lang.FnFunc%d {\n"+
				"version := vr.RootVersion()\n"+
				"fn := checkDerefVar(vr)\n",
			arity, arity)
		fmt.Fprintf(&g.aotDeclarations,
			"if direct, ok := fn.(lang.FnFunc%d); ok {\n"+
				"return func(%s) any {\n"+
				"if vr.RootVersion() == version { return direct(%s) }\n"+
				"return lang.Apply%d(checkDerefVar(vr)%s)\n"+
				"}\n"+
				"}\n",
			arity, paramList, argList, arity, aotAdapterArgs(args))
		fmt.Fprintf(&g.aotDeclarations,
			"if fixed, ok := fn.(lang.FixedArityFn%d); ok {\n"+
				"return func(%s) any {\n"+
				"if vr.RootVersion() == version { return fixed.Invoke%d(%s) }\n"+
				"return lang.Apply%d(checkDerefVar(vr)%s)\n"+
				"}\n"+
				"}\n",
			arity, paramList, arity, argList, arity, aotAdapterArgs(args))
		fmt.Fprintf(&g.aotDeclarations,
			"return func(%s) any {\n"+
				"if vr.RootVersion() == version { return lang.Apply%d(fn%s) }\n"+
				"return lang.Apply%d(checkDerefVar(vr)%s)\n"+
				"}\n"+
				"}\n\n",
			paramList,
			arity, aotAdapterArgs(args),
			arity, aotAdapterArgs(args))
	}
	for arity, used := range linkedArities {
		if !used {
			continue
		}
		params := make([]string, arity)
		args := make([]string, arity)
		for i := range params {
			args[i] = fmt.Sprintf("p%d", i)
			params[i] = args[i] + " any"
		}
		paramList := strings.Join(params, ", ")
		argList := strings.Join(args, ", ")
		fmt.Fprintf(&g.aotDeclarations,
			"func aotLinkFn%d(vr *lang.Var) lang.FnFunc%d {\n"+
				"if vr.IsBound() { return aotLinkBoundFn%d(vr) }\n"+
				"var once sync.Once\n"+
				"var linked lang.FnFunc%d\n"+
				"return func(%s) any {\n"+
				"if !vr.IsBound() { return lang.Apply%d(checkDerefVar(vr)%s) }\n"+
				"once.Do(func() { linked = aotLinkBoundFn%d(vr) })\n"+
				"return linked(%s)\n"+
				"}\n"+
				"}\n\n"+
				"func aotLinkBoundFn%d(vr *lang.Var) lang.FnFunc%d {\n"+
				"fn := checkDerefVar(vr)\n",
			arity, arity,
			arity,
			arity,
			paramList,
			arity, aotAdapterArgs(args),
			arity,
			argList,
			arity, arity)
		fmt.Fprintf(&g.aotDeclarations,
			"if direct, ok := fn.(lang.FnFunc%d); ok { return direct }\n",
			arity)
		fmt.Fprintf(&g.aotDeclarations,
			"if fixed, ok := fn.(lang.FixedArityFn%d); ok { return fixed.Invoke%d }\n",
			arity, arity)
		fmt.Fprintf(&g.aotDeclarations,
			"return func(%s) any { return lang.Apply%d(fn%s) }\n"+
				"}\n\n",
			paramList, arity, aotAdapterArgs(args))
	}
}

func hasAOTAdapterArity(arities [6]bool) bool {
	for _, used := range arities {
		if used {
			return true
		}
	}
	return false
}

func aotAdapterArgs(args []string) string {
	if len(args) == 0 {
		return ""
	}
	return ", " + strings.Join(args, ", ")
}

func aotSupportsArity(value any, arity int) bool {
	switch arity {
	case 0:
		_, ok := value.(lang.FixedArityFn0)
		return ok
	case 1:
		_, ok := value.(lang.FixedArityFn1)
		return ok
	case 2:
		_, ok := value.(lang.FixedArityFn2)
		return ok
	case 3:
		_, ok := value.(lang.FixedArityFn3)
		return ok
	case 4:
		_, ok := value.(lang.FixedArityFn4)
		return ok
	case 5:
		_, ok := value.(lang.FixedArityFn5)
		return ok
	default:
		return false
	}
}
