// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the Apache License, Version 2.0.
// https://www.apache.org/licenses/LICENSE-2.0
package jaxson

// THE TREE-WALKING INTERPRETER: THE ORACLE. SCHEDULED FOR DELETION.
//
// This is the interpreter as it was before the program was compiled to
// closures (compile.go). It is kept for one purpose: as the oracle that the
// differential tests (compile_oracle_test.go in this package, and
// oracle_test.go in pkg/shaxon) run beside the compiled path, to require the
// same output, the same step total and the same error on every fixture. Nothing
// else uses it. A Machine runs it only when tree mode is switched on
// (UseTreeInterpreter), which the tests do and production code never does.
//
// Delete this file, the evalTree/computeTree/runTree dispatch in machine.go,
// the Apply members of the operator table, the Exec members of the core
// instructions, the tree-mode switch, and the two oracle test files, once the
// compiled path has been through a release with no divergence found. Until
// then do not "improve" this file: its value is that it is the old code.

import (
	"strings"
)

func (m *Machine) evalTree(x any) any {
	if mp, ok := x.(map[string]any); ok {
		if d, found := formOf(m.forms, mp); found {
			return d.Eval(m, mp)
		}
	}
	if mp, ok := x.(map[string]any); ok && len(mp) == 1 {
		for k, v := range mp {
			switch k {
			case "$path":
				return m.read(v.([]any))
			case "$lit":
				return Data(v)
			case "$compute":
				return m.compute(v.(map[string]any))
			case "$tpl":
				return m.tpl(v)
			}
		}
	}
	return Data(x)
}

func (m *Machine) tpl(v any) any {
	switch t := v.(type) {
	case map[string]any:
		if len(t) == 1 {
			for k, a := range t {
				if k == "$opt" {
					r, s := m.segs(a.([]any))
					val, e := m.walk(r, s)
					if e != nil {
						if e.Code == "MISSING_PATH" {
							return omit
						}
						panic(e)
					}
					if r == "input" || r == "local" {
						return val
					}
					return Clone(val)
				}
				if strings.HasPrefix(k, "$") {
					return m.eval(t)
				}
			}
		}
		// Members are evaluated in sorted key order. The original interpreter
		// ranged over the map, so which member failed first, and how many
		// steps had been charged by then, depended on Go's random iteration
		// order; the compiled path sorts, and the oracle must agree with it
		// to be a usable comparison.
		out := NewObject(len(t))
		for _, k := range SortedKeys(t) {
			r := m.tpl(t[k])
			if _, o := r.(omitT); o {
				continue
			}
			out.Set(k, r)
		}
		return out
	case []any:
		out := make([]any, 0, len(t))
		for _, a := range t {
			r := m.tpl(a)
			if _, o := r.(omitT); o {
				continue
			}
			out = append(out, r)
		}
		return out
	}
	return v
}

func (m *Machine) computeTree(c map[string]any) any {
	env := map[string]any{}
	if w, ok := c["with"].(map[string]any); ok {
		for _, n := range SortedKeys(w) {
			env[n] = m.eval(w[n])
		}
	}
	return m.expr(c["expr"], env)
}

func (m *Machine) expr(e any, env map[string]any) any {
	switch t := e.(type) {
	case map[string]any:
		return env[t["$v"].(string)]
	case []any:
		op := t[0].(string)
		m.Charge(m.costs.lookup("operator", op, m.costs.Op).At(0))
		return m.apply(t[0].(string), t[1:], env)
	}
	return e
}

func (m *Machine) apply(op string, a []any, env map[string]any) any {
	def, ok := m.operators[op]
	if !ok {
		panic("unreachable op " + op)
	}
	return def.Apply(m, a, env)
}

func (m *Machine) runTree(block []any) {
	for _, raw := range block {
		in := raw.(map[string]any)
		name := in["op"].(string)
		m.Charge(m.costs.lookup("instruction", name, m.costs.Instr).At(0))
		def, ok := m.instructions[name]
		if !ok {
			// CheckProgram/checkBlock already rejects an unknown op
			// before Run ever gets here; this is unreachable in
			// practice, not a new failure mode.
			execFail("TYPE_ERROR", "unknown op %q at execution", in["op"])
		}
		def.Exec(m, in)
	}
}
