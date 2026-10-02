// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the Apache License, Version 2.0.
// https://www.apache.org/licenses/LICENSE-2.0
package jaxson

// Phase 1: the machine type is now Machine (exported, so a host package's
// own runtime can construct one and drive it), carries the instruction
// and operator tables that used to be hardcoded switches (program.go,
// compute.go), and exposes an OnMutate hook — the mutation-path log
// jaxson-shaxon-implementation-plan.md section 1 identifies as the one
// non-cosmetic gap: Shaxon's index-reuse rule needs to know what changed
// since the last build, and this package's own no-aliasing guarantee
// (deep-copy on every read) never had to track that. run()'s dispatch
// loop and apply()'s dispatch are otherwise the same logic as before,
// now driven by table lookups instead of two hardcoded switches.

import (
	"fmt"
	"math/big"
	"strings"
)

// ---------------------------------------------------------------- machine

// Machine is one running instance of a Jaxson program: its three roots,
// its `local.*` bindings, its step counter and limit, and the
// instruction/operator tables in effect. Construct one with NewMachine
// rather than a struct literal — its fields are unexported by design
// (Jaxson's no-ambient-state guarantee extends to not letting an external
// caller reach in and mutate input/state/output directly, only through
// the same set/append/insert/delete instructions a program itself uses).
type Machine struct {
	input, state, output any
	locals               map[string]any
	steps, limit         int
	instructions         map[string]InstructionDef
	operators            map[string]OperatorDef

	// OnMutate, if set, is called after each successful set/append/
	// insert/delete, with the root and the path segments of whatever
	// changed (the array/object actually written to — see program.go's
	// delete cases for exactly what's reported for a deletion). A
	// Jaxson-only caller pays one nil check per mutation for this; it
	// exists so a host package (Shaxon) can build its own mutation-path
	// log without duplicating the four mutation instructions.
	OnMutate func(root string, segs []any)
}

// NewMachine constructs a Machine ready to run a program against the
// given input/state/output roots, with the given step limit. A nil
// instructions or operators table defaults to CoreInstructions() /
// CoreOperators() — the common case for a Jaxson-only caller; a host
// package passes its own extended tables (core plus its own entries).
func NewMachine(input, state, output any, limit int, instructions map[string]InstructionDef, operators map[string]OperatorDef) *Machine {
	if instructions == nil {
		instructions = CoreInstructions()
	}
	if operators == nil {
		operators = CoreOperators()
	}
	return &Machine{
		input: input, state: state, output: output,
		locals: map[string]any{}, limit: limit,
		instructions: instructions, operators: operators,
	}
}

// Step charges one unit against the machine's step limit, failing with a
// RESOURCE_ERROR/STEPS Err if the limit is exceeded. Every instruction
// and every compute-expression application charges one step (see run()
// and expr() below); a host package's own per-element evaluation (a
// shape check, a qualified count) should charge through this same method
// rather than maintaining a second counter, so the two contribute to one
// shared, portable step total.
func (m *Machine) Step() {
	m.steps++
	if m.steps > m.limit {
		fail("RESOURCE_ERROR", "STEPS", "step limit %d exceeded", m.limit)
	}
}

// mutated calls OnMutate if one is set. Every mutation instruction in
// program.go calls this immediately after the mutation succeeds.
func (m *Machine) mutated(root string, segs []any) {
	if m.OnMutate != nil {
		m.OnMutate(root, segs)
	}
}

func (m *Machine) toIndex(r *big.Rat) int {
	if !r.IsInt() || r.Sign() < 0 || r.Num().BitLen() > 31 {
		execFail("BAD_INDEX", "invalid index %s", FormatDecimal(r))
	}
	return int(r.Num().Int64())
}

func (m *Machine) segs(p []any) (string, []any) {
	root := p[0].(string)
	out := make([]any, 0, len(p)-1)
	for _, s := range p[1:] {
		var v any = s
		if mp, ok := s.(map[string]any); ok {
			v = m.eval(mp)
		}
		switch u := v.(type) {
		case string:
			out = append(out, u)
		case *big.Rat:
			out = append(out, m.toIndex(u))
		default:
			execFail("TYPE_ERROR", "a path segment must be a string or an integer, got %s", TypeName(v))
		}
	}
	return root, out
}

func (m *Machine) walk(root string, segs []any) (any, *Err) {
	var cur any
	switch root {
	case "input":
		cur = m.input
	case "state":
		cur = m.state
	case "output":
		cur = m.output
	case "local":
		cur = m.locals[segs[0].(string)]
		segs = segs[1:]
	}
	for _, s := range segs {
		switch k := s.(type) {
		case string:
			mp, ok := cur.(map[string]any)
			if !ok {
				return nil, &Err{"EXECUTION_ERROR", "TYPE_ERROR", fmt.Sprintf("cannot read member %q of %s", k, TypeName(cur))}
			}
			v, ok := mp[k]
			if !ok {
				return nil, &Err{"EXECUTION_ERROR", "MISSING_PATH", fmt.Sprintf("member %q not found", k)}
			}
			cur = v
		case int:
			ar, ok := cur.([]any)
			if !ok {
				return nil, &Err{"EXECUTION_ERROR", "TYPE_ERROR", fmt.Sprintf("cannot index %s", TypeName(cur))}
			}
			if k >= len(ar) {
				return nil, &Err{"EXECUTION_ERROR", "MISSING_PATH", fmt.Sprintf("index %d out of range", k)}
			}
			cur = ar[k]
		}
	}
	return cur, nil
}

func (m *Machine) getAt(root string, segs []any) any {
	v, e := m.walk(root, segs)
	if e != nil {
		panic(e)
	}
	return v
}

func (m *Machine) putAt(root string, segs []any, v any) {
	if len(segs) == 0 {
		switch root {
		case "state":
			m.state = v
		case "output":
			m.output = v
		}
		return
	}
	par := m.getAt(root, segs[:len(segs)-1])
	switch k := segs[len(segs)-1].(type) {
	case string:
		mp, ok := par.(map[string]any)
		if !ok {
			execFail("TYPE_ERROR", "cannot set member %q on %s", k, TypeName(par))
		}
		mp[k] = v
	case int:
		ar, ok := par.([]any)
		if !ok {
			execFail("TYPE_ERROR", "cannot set an index on %s", TypeName(par))
		}
		if k >= len(ar) {
			execFail("BAD_INDEX", "index %d out of range", k)
		}
		ar[k] = v
	}
}

func (m *Machine) read(p []any) any {
	r, s := m.segs(p)
	return Clone(m.getAt(r, s))
}

func (m *Machine) eval(x any) any {
	if mp, ok := x.(map[string]any); ok && len(mp) == 1 {
		for k, v := range mp {
			switch k {
			case "$path":
				return m.read(v.([]any))
			case "$lit":
				return Clone(v)
			case "$compute":
				return Clone(m.compute(v.(map[string]any)))
			case "$tpl":
				return m.tpl(v)
			}
		}
	}
	return Clone(x)
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
					return Clone(val)
				}
				if strings.HasPrefix(k, "$") {
					return m.eval(t)
				}
			}
		}
		out := map[string]any{}
		for k, a := range t {
			r := m.tpl(a)
			if _, o := r.(omitT); o {
				continue
			}
			out[k] = r
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
	return Clone(v)
}

func (m *Machine) compute(c map[string]any) any {
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
		m.Step()
		return m.apply(t[0].(string), t[1:], env)
	}
	return e
}

func num(x any) *big.Rat {
	r, ok := x.(*big.Rat)
	if !ok {
		execFail("TYPE_ERROR", "expected number, got %s", TypeName(x))
	}
	return r
}
func str(x any) string {
	s, ok := x.(string)
	if !ok {
		execFail("TYPE_ERROR", "expected string, got %s", TypeName(x))
	}
	return s
}
func boo(x any) bool {
	b, ok := x.(bool)
	if !ok {
		execFail("TYPE_ERROR", "expected boolean, got %s", TypeName(x))
	}
	return b
}
func integer(x any) *big.Int {
	r := num(x)
	if !r.IsInt() {
		execFail("TYPE_ERROR", "expected integer, got %s", FormatDecimal(r))
	}
	return r.Num()
}
func scaleOf(x any) int {
	r := num(x)
	if !r.IsInt() || r.Sign() < 0 || r.Cmp(big.NewRat(maxDigits, 1)) > 0 {
		execFail("BAD_SCALE", "scale must be an integer from 0 to %d", maxDigits)
	}
	return int(r.Num().Int64())
}

func (m *Machine) apply(op string, a []any, env map[string]any) any {
	def, ok := m.operators[op]
	if !ok {
		panic("unreachable op " + op)
	}
	return def.Apply(m, a, env)
}

func lookup(c, k any) (any, bool) {
	switch t := c.(type) {
	case map[string]any:
		v, ok := t[str(k)]
		return v, ok
	case []any:
		i := new(Machine).toIndex(num(k))
		if i < len(t) {
			return t[i], true
		}
		return nil, false
	}
	execFail("TYPE_ERROR", "cannot look up in %s", TypeName(c))
	return nil, false
}

func (m *Machine) run(block []any) {
	for _, raw := range block {
		in := raw.(map[string]any)
		m.Step()
		def, ok := m.instructions[in["op"].(string)]
		if !ok {
			// CheckProgram/checkBlock already rejects an unknown op
			// before Run ever gets here; this is unreachable in
			// practice, not a new failure mode.
			execFail("TYPE_ERROR", "unknown op %q at execution", in["op"])
		}
		def.Exec(m, in)
	}
}

// RunProgram executes a top-level program block, translating the halt
// instruction's control-flow panic into a normal return (any other panic
// still propagates, to be turned into an *Err by the caller's own
// recover — see the package-level Run below). A host package's own Run
// calls this once it has constructed a Machine with its extended
// instruction table, instead of duplicating this halt-handling.
func (m *Machine) RunProgram(block []any) {
	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(haltSignal); !ok {
				panic(r)
			}
		}
	}()
	m.run(block)
}

// ---------------------------------------------------------------- exported machine primitives

// Walk reads a value at root+segs without cloning it, returning a
// documented Err instead of panicking on a missing path or a type
// mismatch. Exported so a host package's own read-only evaluation (a
// shape check, a target resolution) can walk the same three roots this
// package already knows how to walk, without re-implementing path
// traversal. Unlike GetAt, this never panics on a missing path.
func (m *Machine) Walk(root string, segs []any) (any, *Err) { return m.walk(root, segs) }

// GetAt reads a value at root+segs, panicking with an *Err (recoverable
// the same way Run's own panics are) if the path doesn't resolve. This is
// the same primitive set/append/insert/delete use internally.
func (m *Machine) GetAt(root string, segs []any) any { return m.getAt(root, segs) }

// Eval evaluates an operand — a `$path`/`$lit`/`$compute`/`$tpl` form, or
// a plain literal — exactly as a `set`/`append`/`if`/... instruction's
// own `value`/`cond`/... field would be evaluated. Exported so a host
// instruction's Exec (Shaxon's "check", in particular) can evaluate its
// own operand fields using the same machinery, and so a host's shape/
// field evaluation can resolve a `$path` focus-node reference the same
// way the core language does.
func (m *Machine) Eval(x any) any { return m.eval(x) }

// RunCompute runs a `$compute` island — `{"with": {...}, "expr": [...]}`
// — and returns its result, charging steps exactly as an inline
// `$compute` embedded in a core instruction's operand would. Exported so
// Shaxon's "check" instruction (a `$compute` bound to `local.focus`) and
// named-compute-reuse machinery can run a compute island without
// re-implementing expression evaluation.
func (m *Machine) RunCompute(c map[string]any) any { return m.compute(c) }
