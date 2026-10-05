// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the Apache License, Version 2.0.
// https://www.apache.org/licenses/LICENSE-2.0
package jaxson

// Compiled execution. A program (or an operand, or a `$compute` island) is
// translated once into a tree of Go closures, and the closures are what run.
// What the tree-walking interpreter (tree_oracle.go) did again on every
// evaluation is done once here:
//
//   - finding out which `$`-form an operand is (formerly a map iteration);
//   - looking up the instruction, the operator and their costs by name;
//   - converting path segments to strings and array indices;
//   - binding `with` names to slots, so `$v` is an index, not a map lookup.
//
// This is the technique iual uses for ual's compute blocks (threaded code:
// flat slices of closures over variable slots), with one difference: ual's
// slots are typed, and Jaxson's values are dynamic, so a closure here returns
// an `any`.
//
// Nothing about the language changes. The compiled program charges the same
// steps in the same order, raises the same errors, and reports the same
// mutations as the interpreter; compile_oracle_test.go and oracle_test.go
// (pkg/shaxon) run both and require it.
//
// A node the compiler does not know how to compile (a host-registered
// operand form, a host-registered instruction without a Compile function)
// is run through its own Eval or Exec, exactly as before.

import (
	"math/big"
	"reflect"
	"sort"
	"strings"
)

// treeDefault is the interpreter a new Machine uses. See UseTreeInterpreter.
var treeDefault bool

// UseTreeInterpreter switches Machines created afterwards to the
// tree-walking interpreter (tree_oracle.go) when on is true. It exists only
// for the differential tests that compare the two; it is not safe to call
// while Machines are running, and production code has no reason to.
//
// DEPRECATED: scheduled for deletion together with the oracle.
func UseTreeInterpreter(on bool) { treeDefault = on }

// Compiler compiles program text for one Machine. A host instruction's
// Compile function receives one, and uses its methods to compile the
// operand, path and block fields of the instruction it is compiling.
type Compiler struct{ m *Machine }

// Operand compiles an operand: a `$path`, `$lit`, `$compute`, `$tpl`, a
// registered form, or a plain literal.
func (c *Compiler) Operand(x any) func(*Machine) any {
	mp, ok := x.(map[string]any)
	if !ok {
		lit := Data(x) // a literal array may hold objects
		return func(*Machine) any { return lit }
	}
	if d, found := formOf(c.m.forms, mp); found {
		return func(m *Machine) any { return d.Eval(m, mp) }
	}
	if len(mp) == 1 {
		for k, v := range mp {
			switch k {
			case "$path":
				return c.pathRead(v.([]any))
			case "$lit":
				lit := Data(v)
				return func(*Machine) any { return lit }
			case "$compute":
				return c.compute(v.(map[string]any))
			case "$tpl":
				return c.tpl(v)
			}
		}
	}
	lit := Data(mp)
	return func(*Machine) any { return lit }
}

// Path compiles a path (a segment array whose segments may be operands) to a
// function returning the root and the evaluated segments, as Machine.segs
// does, except that a member name known at compile time is a Key and not a
// string (Machine.Walk, GetAt and the mutation instructions take either).
// The returned slice may be shared between calls: callers must not change it.
func (c *Compiler) Path(p []any) func(*Machine) (string, []any) {
	return c.path(p, false)
}

// path compiles a path. With scratch set, a path that has to be built at run
// time is built in the machine's scratch stack instead of on the heap: the
// caller then owns it only until it releases the stack back to the level it
// had before calling (see pathRead), so it must not keep or return it.
func (c *Compiler) path(p []any, scratch bool) func(*Machine) (string, []any) {
	root := p[0].(string)
	segs := p[1:]
	fixed, static := c.fixedSegs(segs)
	if root == "local" && static && len(fixed) > 0 {
		fixed[0] = fixed[0].(Key).Value() // the binding's name is looked up by string
	}
	if static {
		return func(*Machine) (string, []any) { return root, fixed }
	}
	type seg struct {
		lit any
		op  func(*Machine) any
	}
	plan := make([]seg, len(segs))
	for i, sg := range segs {
		if mp, ok := sg.(map[string]any); ok {
			plan[i] = seg{op: c.Operand(mp)}
		} else if s, isStr := sg.(string); isStr && !(root == "local" && i == 0) {
			plan[i] = seg{lit: MakeKey(s)}
		} else {
			plan[i] = seg{lit: sg}
		}
	}
	return func(m *Machine) (string, []any) {
		var out []any
		if scratch {
			out = m.alloc(len(plan))
		} else {
			out = make([]any, len(plan))
		}
		for i, s := range plan {
			v := s.lit
			if s.op != nil {
				v = s.op(m)
			}
			switch u := v.(type) {
			case Key, string:
				out[i] = v // already boxed: no new allocation
			case *big.Rat:
				out[i] = boxInt(m.toIndex(u))
			default:
				execFail("TYPE_ERROR", "a path segment must be a string or an integer, got %s", TypeName(v))
			}
		}
		return root, out
	}
}

// fixedSegs converts path segments that are known now (names and literal
// indices) and reports whether they all are.
func (c *Compiler) fixedSegs(segs []any) (fixed []any, static bool) {
	fixed = make([]any, len(segs))
	static = true
	for i, sg := range segs {
		switch t := sg.(type) {
		case string:
			fixed[i] = MakeKey(t)
		case *big.Rat:
			if idx, ok := c.m.tryIndex(t); ok {
				fixed[i] = idx
			} else {
				static = false // raised at run time, as before
			}
		default:
			static = false
		}
	}
	return
}

// Block compiles a block (an array of instructions).
func (c *Compiler) Block(b any) func(*Machine) {
	arr, ok := b.([]any)
	if !ok {
		return func(*Machine) {}
	}
	fns := make([]func(*Machine), len(arr))
	for i, raw := range arr {
		fns[i] = c.instruction(raw.(map[string]any))
	}
	if len(fns) == 1 {
		return fns[0]
	}
	return func(m *Machine) {
		for _, f := range fns {
			f(m)
		}
	}
}

func (c *Compiler) instruction(in map[string]any) func(*Machine) {
	name := in["op"].(string)
	cost, lazy := c.m.instrCost(name)
	def, known := c.m.instructions[name]
	var body func(*Machine)
	switch {
	case !known:
		body = func(m *Machine) { execFail("TYPE_ERROR", "unknown op %q at execution", in["op"]) }
	case def.Compile != nil:
		body = def.Compile(c, in)
	default:
		body = func(m *Machine) { def.Exec(m, in) }
	}
	if lazy {
		return func(m *Machine) {
			m.Charge(m.costs.lookup("instruction", name, m.costs.Instr).At(0))
			body(m)
		}
	}
	return func(m *Machine) {
		m.Charge(cost)
		body(m)
	}
}

func (m *Machine) instrCost(name string) (cost int64, lazy bool) {
	defer func() {
		if recover() != nil {
			lazy = true
		}
	}()
	return m.costs.lookup("instruction", name, m.costs.Instr).At(0), false
}

func (m *Machine) opCost(name string) (cost int64, lazy bool) {
	defer func() {
		if recover() != nil {
			lazy = true
		}
	}()
	return m.costs.lookup("operator", name, m.costs.Op).At(0), false
}

func (m *Machine) tryIndex(r *big.Rat) (idx int, ok bool) {
	defer func() {
		if recover() != nil {
			ok = false
		}
	}()
	return m.toIndex(r), true
}

// ---------------------------------------------------------------- paths

func (c *Compiler) pathRead(p []any) func(*Machine) any {
	root := p[0].(string)
	if root == "local" {
		if fixed, static := c.fixedSegs(p[1:]); static && len(fixed) > 0 {
			name, rest := fixed[0].(Key).Value(), fixed[1:]
			if len(rest) == 0 {
				return func(m *Machine) any { return m.local(name) }
			}
			return func(m *Machine) any {
				v, e := walkSegs(m.local(name), rest)
				if e != nil {
					panic(e)
				}
				return v
			}
		}
	}
	plan := c.path(p, true)
	if root == "input" || root == "local" {
		return func(m *Machine) any {
			base := m.sp
			r, s := plan(m)
			v := m.getAt(r, s)
			m.release(base)
			return v
		}
	}
	return func(m *Machine) any {
		base := m.sp
		r, s := plan(m)
		v := m.getAt(r, s)
		m.release(base)
		return Clone(v)
	}
}

// smallInts holds pre-boxed ints, so that a computed index below 1024 does
// not allocate when it is stored in a path segment.
var smallInts = func() (t [1024]any) {
	for i := range t {
		t[i] = i
	}
	return
}()

func boxInt(i int) any {
	if i >= 0 && i < len(smallInts) {
		return smallInts[i]
	}
	return i
}

// ---------------------------------------------------------------- $tpl

func (c *Compiler) tpl(v any) func(*Machine) any {
	switch t := v.(type) {
	case map[string]any:
		if len(t) == 1 {
			for k, a := range t {
				if k == "$opt" {
					plan := c.path(a.([]any), true)
					return func(m *Machine) any {
						base := m.sp
						r, s := plan(m)
						val, e := m.walk(r, s)
						m.release(base)
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
				}
				if strings.HasPrefix(k, "$") {
					return c.Operand(t)
				}
			}
		}
		names := make([]string, 0, len(t))
		for k := range t {
			names = append(names, k)
		}
		sort.Strings(names)
		fns := make([]func(*Machine) any, len(names))
		keys := make([]Key, len(names))
		for i, k := range names {
			fns[i] = c.tpl(t[k])
			keys[i] = MakeKey(k)
		}
		return func(m *Machine) any {
			out := NewObject(len(keys))
			for i, k := range keys {
				r := fns[i](m)
				if _, o := r.(omitT); o {
					continue
				}
				out.add(k, r)
			}
			return out
		}
	case []any:
		fns := make([]func(*Machine) any, len(t))
		for i, a := range t {
			fns[i] = c.tpl(a)
		}
		return func(m *Machine) any {
			out := make([]any, 0, len(fns))
			for _, f := range fns {
				r := f(m)
				if _, o := r.(omitT); o {
					continue
				}
				out = append(out, r)
			}
			return out
		}
	}
	return func(*Machine) any { return v }
}

// ---------------------------------------------------------------- $compute

// exprFn is a compiled compute expression. env holds the values of the
// island's `with` bindings, by slot.
type exprFn func(m *Machine, env []any) any

func (c *Compiler) compute(cm map[string]any) func(*Machine) any {
	slots := map[string]int{}
	var withs []func(*Machine) any
	if w, ok := cm["with"].(map[string]any); ok {
		for i, n := range SortedKeys(w) {
			slots[n] = i
			withs = append(withs, c.Operand(w[n]))
		}
	}
	ex, ok := c.expr(cm["expr"], slots)
	if !ok {
		// An operator this compiler does not know (a host's own): run the
		// island through the interpreter.
		return func(m *Machine) any { return m.computeTree(cm) }
	}
	n := len(withs)
	return func(m *Machine) any {
		base := m.sp
		env := m.alloc(n)
		for i, w := range withs {
			env[i] = w(m)
		}
		r := ex(m, env)
		m.release(base)
		return r
	}
}

func (c *Compiler) expr(e any, slots map[string]int) (exprFn, bool) {
	switch t := e.(type) {
	case map[string]any:
		name := t["$v"].(string)
		if i, ok := slots[name]; ok {
			return func(_ *Machine, env []any) any { return env[i] }, true
		}
		return func(*Machine, []any) any { return nil }, true
	case []any:
		op := t[0].(string)
		d, known := c.m.operators[op]
		if !known {
			return func(*Machine, []any) any { panic("unreachable op " + op) }, true
		}
		cost, lazy := c.m.opCost(op)
		charge := func(m *Machine) {
			if lazy {
				m.Charge(m.costs.lookup("operator", op, m.costs.Op).At(0))
			} else {
				m.Charge(cost)
			}
		}
		args := make([]exprFn, len(t)-1)
		for i, a := range t[1:] {
			f, ok := c.expr(a, slots)
			if !ok {
				return nil, false
			}
			args[i] = f
		}
		switch op {
		case "and":
			return func(m *Machine, env []any) any {
				charge(m)
				for _, a := range args {
					if !boo(a(m, env)) {
						return false
					}
				}
				return true
			}, true
		case "or":
			return func(m *Machine, env []any) any {
				charge(m)
				for _, a := range args {
					if boo(a(m, env)) {
						return true
					}
				}
				return false
			}, true
		case "select":
			return func(m *Machine, env []any) any {
				charge(m)
				if boo(args[0](m, env)) {
					return args[1](m, env)
				}
				return args[2](m, env)
			}, true
		}
		if d.Fn == nil {
			return nil, false
		}
		fn := d.Fn
		n := len(args)
		return func(m *Machine, env []any) any {
			charge(m)
			base := m.sp
			v := m.alloc(n)
			for i, a := range args {
				v[i] = a(m, env)
			}
			r := fn(m, v)
			m.release(base)
			return r
		}, true
	}
	return func(*Machine, []any) any { return e }, true
}

// ---------------------------------------------------------------- scratch

// scratchSize is the number of value slots a Machine keeps for the `with`
// environments and operator arguments of the islands being evaluated. When
// nesting exhausts it, alloc falls back to the heap.
const scratchSize = 1024

func (m *Machine) alloc(n int) []any {
	if n == 0 {
		return nil
	}
	if m.scratch == nil {
		m.scratch = make([]any, scratchSize)
	}
	if m.sp+n <= len(m.scratch) {
		s := m.scratch[m.sp : m.sp+n : m.sp+n]
		m.sp += n
		return s
	}
	return make([]any, n)
}

// release frees everything allocated since sp was base.
func (m *Machine) release(base int) {
	if m.sp > base {
		clear(m.scratch[base:m.sp])
		m.sp = base
	}
}

// ---------------------------------------------------------------- the machine's entry points

// eval evaluates an operand. In compiled mode an operand reached from outside
// a compiled program (a host instruction's Exec, an index key) is compiled
// the first time it is seen and kept, keyed by the identity of its map; the
// entry keeps the map alive, so its address cannot be reused.
func (m *Machine) eval(x any) any {
	if m.tree {
		return m.evalTree(x)
	}
	mp, ok := x.(map[string]any)
	if !ok {
		return x
	}
	k := reflect.ValueOf(mp).Pointer()
	if e, ok := m.cache[k]; ok {
		return e.fn(m)
	}
	fn := (&Compiler{m}).Operand(mp)
	if m.cache == nil {
		m.cache = map[uintptr]cacheEntry{}
	}
	m.cache[k] = cacheEntry{key: mp, fn: fn}
	return fn(m)
}

// compute runs a `$compute` island, compiled once per island.
func (m *Machine) compute(c map[string]any) any {
	if m.tree {
		return m.computeTree(c)
	}
	k := reflect.ValueOf(c).Pointer()
	if e, ok := m.ccache[k]; ok {
		return e.fn(m)
	}
	fn := (&Compiler{m}).compute(c)
	if m.ccache == nil {
		m.ccache = map[uintptr]cacheEntry{}
	}
	m.ccache[k] = cacheEntry{key: c, fn: fn}
	return fn(m)
}

// run executes a block of instructions. A block is compiled each time it is
// run; the program is run once, and the small blocks a host synthesises are
// not worth keeping.
func (m *Machine) run(block []any) {
	if m.tree {
		m.runTree(block)
		return
	}
	(&Compiler{m}).Block(block)(m)
}

type cacheEntry struct {
	key map[string]any
	fn  func(*Machine) any
}
