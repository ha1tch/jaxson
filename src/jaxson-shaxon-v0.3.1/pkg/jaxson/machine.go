// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the GNU General Public License, version 3.
// https://www.gnu.org/licenses/gpl-3.0.html
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
)

// ---------------------------------------------------------------- locals

// localVar is one `local.<name>` binding. The bindings in scope form a stack,
// innermost last: there are few (a loop variable, its index, a host's focus),
// so finding one by scanning from the top costs less than hashing its name.
type localVar struct {
	name string
	v    any
}

// local returns the innermost binding of name, or nil if there is none.
func (m *Machine) local(name string) any {
	for i := len(m.locals) - 1; i >= 0; i-- {
		if m.locals[i].name == name {
			return m.locals[i].v
		}
	}
	return nil
}

// dropLocals pops the stack back to n bindings.
func (m *Machine) dropLocals(n int) {
	if len(m.locals) > n {
		clear(m.locals[n:])
		m.locals = m.locals[:n]
	}
}

// smallRats holds the first indices as shared numbers, so that a loop's index
// variable allocates nothing. Numbers are immutable.
var smallRats = func() (t [1024]*big.Rat) {
	for i := range t {
		t[i] = big.NewRat(int64(i), 1)
	}
	return
}()

func ratInt(i int) *big.Rat {
	if i >= 0 && i < len(smallRats) {
		return smallRats[i]
	}
	return big.NewRat(int64(i), 1)
}

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
	locals               []localVar // a stack: the innermost binding of a name is the last
	steps, limit         int64
	costs                CostTable
	instructions         map[string]InstructionDef
	operators            map[string]OperatorDef
	forms                map[string]FormDef

	// Compiled execution (compile.go). tree selects the tree-walking
	// oracle instead; cache and ccache hold operands and `$compute`
	// islands compiled on first use; scratch and sp are the value slots
	// the islands being evaluated borrow.
	tree          bool
	cache, ccache map[uintptr]cacheEntry
	scratch       []any
	sp            int

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
		input: Data(input), state: Data(state), output: Data(output),
		locals: make([]localVar, 0, 8), limit: int64(limit), costs: UnitTable(),
		instructions: instructions, operators: operators,
		tree: treeDefault,
	}
}

// Step charges one unit against the machine's step limit, failing with a
// RESOURCE_ERROR/STEPS Err if the limit is exceeded. Every instruction
// and every compute-expression application charges one step (see run()
// and expr() below); a host package's own per-element evaluation (a
// shape check, a qualified count) should charge through this same method
// rather than maintaining a second counter, so the two contribute to one
// shared, portable step total.
func (m *Machine) Step() { m.Charge(1) }

// mutated calls OnMutate if one is set. Every mutation instruction in
// program.go calls this immediately after the mutation succeeds.
func (m *Machine) mutated(root string, segs []any) {
	if m.OnMutate != nil {
		m.OnMutate(root, plainSegs(segs))
	}
}

// plainSegs returns segs with every Key turned back into its string, which is
// what a host's mutation log understands. A path with no Key is returned as
// it is.
func plainSegs(segs []any) []any {
	var out []any
	for i, s := range segs {
		if k, ok := s.(Key); ok {
			if out == nil {
				out = make([]any, len(segs))
				copy(out, segs[:i])
			}
			out[i] = k.Value()
		} else if out != nil {
			out[i] = s
		}
	}
	if out == nil {
		return segs
	}
	return out
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
		cur = m.local(segs[0].(string))
		segs = segs[1:]
	}
	return walkSegs(cur, segs)
}

// walkSegs follows segs from cur.
func walkSegs(cur any, segs []any) (any, *Err) {
	for _, s := range segs {
		switch k := s.(type) {
		case Key: // a name known when the program was compiled: the quick path
			o, ok := cur.(*Object)
			if !ok {
				return nil, &Err{"EXECUTION_ERROR", "TYPE_ERROR", fmt.Sprintf("cannot read member %q of %s", k.Value(), TypeName(cur))}
			}
			v, ok := o.GetKey(k)
			if !ok {
				return nil, &Err{"EXECUTION_ERROR", "MISSING_PATH", fmt.Sprintf("member %q not found", k.Value())}
			}
			cur = v
		case string:
			o, ok := cur.(*Object)
			if !ok {
				return nil, &Err{"EXECUTION_ERROR", "TYPE_ERROR", fmt.Sprintf("cannot read member %q of %s", k, TypeName(cur))}
			}
			v, ok := o.Get(k)
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
	case Key:
		o, ok := par.(*Object)
		if !ok {
			execFail("TYPE_ERROR", "cannot set member %q on %s", k.Value(), TypeName(par))
		}
		o.SetKey(k, v)
	case string:
		o, ok := par.(*Object)
		if !ok {
			execFail("TYPE_ERROR", "cannot set member %q on %s", k, TypeName(par))
		}
		o.Set(k, v)
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

// read evaluates a $path. Values are shared, not copied, when they cannot
// change under the reader: everything in `input` and `local` is immutable (no
// instruction writes to either root), and a *big.Rat, string, boolean or null
// is immutable wherever it lives. A composite read from `state` or `output`
// is deep-copied, so a loop over it iterates a snapshot. Whatever is stored
// into state or output is copied by the storing instruction (own), so no
// value in a mutable root ever aliases a shared one.
func (m *Machine) read(p []any) any {
	r, s := m.segs(p)
	v := m.getAt(r, s)
	if r == "input" || r == "local" {
		return v
	}
	return Clone(v)
}

// own returns a value the caller may store in state or output: a deep copy
// of v, which may be shared with the input, a local, or the program text.
func (m *Machine) own(v any) any { return Clone(v) }

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

func lookup(c, k any) (any, bool) {
	switch t := c.(type) {
	case *Object:
		return t.Get(str(k))
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

// RunProgram executes a top-level program block, translating the halt
// instruction's control-flow panic into a normal return (any other panic
// still propagates, to be turned into an *Err by the caller's own
// recover — see the package-level Run below). A host package's own Run
// calls this once it has constructed a Machine with its extended
// instruction table, instead of duplicating this halt-handling.
func (m *Machine) RunProgram(block []any) {
	sp := m.sp
	nl := len(m.locals)
	defer func() {
		m.release(sp)
		m.dropLocals(nl)
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
//
// The result may share structure with the machine's input, a local, or the
// program text, so it must be treated as read-only. A host that wants to keep
// or change it copies it first (Clone).
func (m *Machine) Eval(x any) any {
	sp, nl := m.sp, len(m.locals)
	defer func() { m.release(sp); m.dropLocals(nl) }()
	return m.eval(x)
}

// RunCompute runs a `$compute` island — `{"with": {...}, "expr": [...]}`
// — and returns its result, charging steps exactly as an inline
// `$compute` embedded in a core instruction's operand would. Exported so
// Shaxon's "check" instruction (a `$compute` bound to `local.focus`) and
// named-compute-reuse machinery can run a compute island without
// re-implementing expression evaluation.
func (m *Machine) RunCompute(c map[string]any) any {
	sp, nl := m.sp, len(m.locals)
	defer func() { m.release(sp); m.dropLocals(nl) }()
	return m.compute(c)
}
