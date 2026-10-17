//go:build !glj_aot_runtime

package runtime

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
)

func (g *Generator) generateCoreNativeLinks() string {
	if g.aotNamespace.Name().String() != "clojure.core" {
		return ""
	}
	targets := make([]*aotSpecializationTarget, 0, len(g.aotCallTargets))
	for _, target := range g.aotCallTargets {
		if !target.arityDispatch && target.arity > 5 {
			continue
		}
		targets = append(targets, target)
	}
	sort.Slice(targets, func(i, j int) bool { return targets[i].directFnVar < targets[j].directFnVar })
	var setup, body bytes.Buffer
	body.WriteString("runtime.RegisterCoreNativeLinks(func() {\n")
	for _, target := range targets {
		vr := target.vr
		id := g.allocVarVar(vr.Namespace().Name().String(), vr.Symbol().String())
		version := "nativeVersion" + target.directFnVar
		fmt.Fprintf(&setup, "%s := %s.RootVersion()\n", version, id)
		fmt.Fprintf(&body, "if %s.RootVersion() != %s && runtime.IsDefaultCoreVar(%s) {\n", id, version, id)
		body.WriteString("value := checkDerefVar(" + id + ")\n")
		if target.arityDispatch {
			fmt.Fprintf(&body, "%s = runtime.CoreArityFn(value.(lang.IFn))\n", target.directFnVar)
			for arity, slot := range target.directArityVars {
				if slot != "" {
					writeCoreNativeSlot(&body, slot, arity)
				}
			}
		} else {
			writeCoreNativeSlot(&body, target.directFnVar, target.arity)
		}
		if target.rootVersionVar != "" {
			fmt.Fprintf(&body, "%s = %s.RootVersion()\n", target.rootVersionVar, id)
		}
		body.WriteString("}\n")
	}
	body.WriteString("})\n")
	return setup.String() + body.String()
}

func writeCoreNativeSlot(output *bytes.Buffer, slot string, arity int) {
	parameters, arguments := make([]string, arity), make([]string, arity)
	for i := range parameters {
		arguments[i] = fmt.Sprintf("p%d", i)
		parameters[i] = arguments[i] + " any"
	}
	suffix := ""
	if len(arguments) != 0 {
		suffix = ", " + strings.Join(arguments, ", ")
	}
	if arity == 0 || arity >= 5 {
		fmt.Fprintf(output, "%s = lang.FnFunc%d(func(%s) any { return lang.Apply%d(value%s) })\n",
			slot, arity, strings.Join(parameters, ", "), arity, suffix)
		return
	}
	fmt.Fprintf(output, "if direct, ok := lang.DirectFn%d(value); ok { %s = direct } else {\n", arity, slot)
	fmt.Fprintf(output, "%s = lang.FnFunc%d(func(%s) any { return lang.Apply%d(value%s) })\n}\n",
		slot, arity, strings.Join(parameters, ", "), arity, suffix)
}
