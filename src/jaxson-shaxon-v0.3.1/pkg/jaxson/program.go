// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the GNU General Public License, version 3.
// https://www.gnu.org/licenses/gpl-3.0.html
package jaxson

// Phase 1: the instruction table. Extracted from what was previously one
// hardcoded switch in checkBlock (static checking) and another in
// machine.go's run() (execution), so a host package can register
// additional instructions without forking either switch. Shaxon adds
// exactly one, "check" (jaxson-shaxon-implementation-plan.md section
// 4). Behaviour of the eight core instructions is unchanged from the
// original jaxrun.go — this is a reshaping of the same logic into data,
// not a rewrite of it.

// Checker carries the state threaded through static checking of a
// program. Locals is which `local.*` names are currently in scope (what
// a bare `map[string]bool` carried before Phase 1); Instructions is the
// table in effect — CoreInstructions() alone, or CoreInstructions() plus
// a host's own entries; Host is an open slot for host-specific state a
// registered instruction's Check function may need. Shaxon's "check"
// instruction uses Host to confirm its `shape` field names something
// declared in the package's `shapes` registry, without this package
// knowing shapes exist.
type Checker struct {
	Locals       map[string]bool
	Instructions map[string]InstructionDef
	Host         any
	Forms        map[string]FormDef // operand forms beyond the core's (forms.go)
}

// clone returns a Checker sharing Instructions and Host with c, but with
// an independent copy of Locals — for entering a nested scope (a `for`
// loop's body) without the new binding leaking back into the parent's.
func (c *Checker) clone() *Checker {
	l := make(map[string]bool, len(c.Locals)+2)
	for k, v := range c.Locals {
		l[k] = v
	}
	return &Checker{Locals: l, Instructions: c.Instructions, Host: c.Host, Forms: c.Forms}
}

// NewChecker builds a Checker with no locals in scope, ready to check a
// top-level program block against the given instruction table.
func NewChecker(instructions map[string]InstructionDef) *Checker {
	return &Checker{Locals: map[string]bool{}, Instructions: instructions}
}

// InstructionDef is one entry in an instruction table: an opcode's
// required and optional fields (used for the generic "missing field" /
// "unknown field" checks every instruction shares, so a host instruction
// gets that validation for free), its static Check, and its Exec.
type InstructionDef struct {
	Name  string
	Req   []string
	Opt   []string
	Check func(in map[string]any, c *Checker)
	Exec  func(m *Machine, in map[string]any)
	// Compile, if set, returns the compiled form of one instruction: a
	// function that does what Exec does, with the instruction's operand,
	// path and block fields compiled once through c. It is optional. An
	// instruction without it is run through Exec, which evaluates its
	// operands with Machine.Eval (compiled on first use and kept).
	Compile func(c *Compiler, in map[string]any) func(*Machine)
}

// The four mutations, shared by the instructions' Exec (the interpreter) and
// their Compile (the compiled path): what is written, and what OnMutate is
// told, is defined once.

func (m *Machine) doSet(r string, s []any, val any) {
	m.putAt(r, s, val)
	m.mutated(r, s)
}

func (m *Machine) doAppend(r string, s []any, val any) {
	ar, ok := m.getAt(r, s).([]any)
	if !ok {
		execFail("TYPE_ERROR", "append target is not an array")
	}
	// Amortised: the array grows in place when it has room. That is safe
	// because every array in state or output is owned by exactly one place
	// (set, append and insert store a private copy, and a composite read
	// from state or output is a copy), so no other holder can see, or
	// append to, the same backing store. A loop that was handed the array
	// earlier holds its own copy (read) or a header whose length does not
	// change. TestAppendDoesNotAlias pins this.
	m.putAt(r, s, append(ar, val))
	m.mutated(r, s)
}

func (m *Machine) doInsert(r string, s []any, at int, val any) {
	ar, ok := m.getAt(r, s).([]any)
	if !ok {
		execFail("TYPE_ERROR", "insert target is not an array")
	}
	if at > len(ar) {
		execFail("BAD_INDEX", "insert position %d beyond length %d", at, len(ar))
	}
	out := make([]any, 0, len(ar)+1)
	out = append(out, ar[:at]...)
	out = append(out, val)
	out = append(out, ar[at:]...)
	m.putAt(r, s, out)
	m.mutated(r, s)
}

func (m *Machine) doDelete(r string, s []any) {
	par := m.getAt(r, s[:len(s)-1])
	last := s[len(s)-1]
	if kk, isKey := last.(Key); isKey {
		last = kk.Value() // a delete is rare enough not to need the quick path
	}
	switch k := last.(type) {
	case string:
		o, ok := par.(*Object)
		if !ok {
			execFail("TYPE_ERROR", "delete member of %s", TypeName(par))
		}
		if !o.Delete(k) {
			execFail("MISSING_PATH", "member %q not found", k)
		}
		// The mutated path reported is the container (the
		// object whose membership changed), not the deleted
		// member's own path — matching the array-delete case
		// below, and simpler for a mutation-log consumer to
		// treat uniformly: "this container's contents
		// changed," not "this specific removed path is now
		// gone."
		m.mutated(r, s[:len(s)-1])
	case int:
		ar, ok := par.([]any)
		if !ok {
			execFail("TYPE_ERROR", "delete index of %s", TypeName(par))
		}
		if k >= len(ar) {
			execFail("BAD_INDEX", "index %d out of range", k)
		}
		out := make([]any, 0, len(ar)-1)
		out = append(out, ar[:k]...)
		out = append(out, ar[k+1:]...)
		m.putAt(r, s[:len(s)-1], out)
		m.mutated(r, s[:len(s)-1])
	}
}

// CoreInstructions returns Jaxson's eight core instructions, unchanged in
// behaviour from the original jaxrun.go. A host package builds its own
// runtime's instruction table as this map plus its own additional
// entries — never by replacing or wrapping these.
func CoreInstructions() map[string]InstructionDef {
	return map[string]InstructionDef{
		"set": {
			Name: "set", Req: []string{"path", "value"},
			Check: func(in map[string]any, c *Checker) {
				checkPath(in["path"], c, true)
				checkOperand(in["value"], c)
			},
			Exec: func(m *Machine, in map[string]any) {
				val := m.own(m.eval(in["value"]))
				r, s := m.segs(in["path"].([]any))
				m.doSet(r, s, val)
			},
			Compile: func(c *Compiler, in map[string]any) func(*Machine) {
				value, path := c.Operand(in["value"]), c.Path(in["path"].([]any))
				return func(m *Machine) {
					val := m.own(value(m))
					r, s := path(m)
					m.doSet(r, s, val)
				}
			},
		},
		"append": {
			Name: "append", Req: []string{"path", "value"},
			Check: func(in map[string]any, c *Checker) {
				checkPath(in["path"], c, true)
				checkOperand(in["value"], c)
			},
			Exec: func(m *Machine, in map[string]any) {
				val := m.own(m.eval(in["value"]))
				r, s := m.segs(in["path"].([]any))
				m.doAppend(r, s, val)
			},
			Compile: func(c *Compiler, in map[string]any) func(*Machine) {
				value, path := c.Operand(in["value"]), c.Path(in["path"].([]any))
				return func(m *Machine) {
					val := m.own(value(m))
					r, s := path(m)
					m.doAppend(r, s, val)
				}
			},
		},
		"insert": {
			Name: "insert", Req: []string{"path", "at", "value"},
			Check: func(in map[string]any, c *Checker) {
				checkPath(in["path"], c, true)
				checkOperand(in["at"], c)
				checkOperand(in["value"], c)
			},
			Exec: func(m *Machine, in map[string]any) {
				val := m.own(m.eval(in["value"]))
				at := m.toIndex(num(m.eval(in["at"])))
				r, s := m.segs(in["path"].([]any))
				m.doInsert(r, s, at, val)
			},
			Compile: func(c *Compiler, in map[string]any) func(*Machine) {
				value, atf, path := c.Operand(in["value"]), c.Operand(in["at"]), c.Path(in["path"].([]any))
				return func(m *Machine) {
					val := m.own(value(m))
					at := m.toIndex(num(atf(m)))
					r, s := path(m)
					m.doInsert(r, s, at, val)
				}
			},
		},
		"delete": {
			Name: "delete", Req: []string{"path"},
			Check: func(in map[string]any, c *Checker) {
				checkPath(in["path"], c, true)
				if len(in["path"].([]any)) < 2 {
					progFail("delete needs a path below a root")
				}
			},
			Exec: func(m *Machine, in map[string]any) {
				r, s := m.segs(in["path"].([]any))
				m.doDelete(r, s)
			},
			Compile: func(c *Compiler, in map[string]any) func(*Machine) {
				path := c.Path(in["path"].([]any))
				return func(m *Machine) {
					r, s := path(m)
					m.doDelete(r, s)
				}
			},
		},
		"if": {
			Name: "if", Req: []string{"cond", "then"}, Opt: []string{"else"},
			Check: func(in map[string]any, c *Checker) {
				checkOperand(in["cond"], c)
				checkBlock(in["then"], c)
				if e, has := in["else"]; has {
					checkBlock(e, c)
				}
			},
			Exec: func(m *Machine, in map[string]any) {
				if boo(m.eval(in["cond"])) {
					m.run(in["then"].([]any))
				} else if e, has := in["else"]; has {
					m.run(e.([]any))
				}
			},
			Compile: func(c *Compiler, in map[string]any) func(*Machine) {
				cond, then := c.Operand(in["cond"]), c.Block(in["then"])
				var els func(*Machine)
				if e, has := in["else"]; has {
					els = c.Block(e)
				}
				return func(m *Machine) {
					if boo(cond(m)) {
						then(m)
					} else if els != nil {
						els(m)
					}
				}
			},
		},
		"for": {
			Name: "for", Req: []string{"in", "as", "do"}, Opt: []string{"index"},
			Check: func(in map[string]any, c *Checker) {
				checkOperand(in["in"], c)
				as, ok := in["as"].(string)
				if !ok || !nameRe.MatchString(as) || c.Locals[as] {
					progFail("for: bad or shadowed binding name")
				}
				inner := c.clone()
				inner.Locals[as] = true
				if ix, has := in["index"]; has {
					is, ok := ix.(string)
					if !ok || !nameRe.MatchString(is) || c.Locals[is] || is == as {
						progFail("for: bad or shadowed index name")
					}
					inner.Locals[is] = true
				}
				checkBlock(in["do"], inner)
			},
			Exec: func(m *Machine, in map[string]any) {
				arr, ok := m.eval(in["in"]).([]any)
				if !ok {
					execFail("TYPE_ERROR", "for: in must be an array")
				}
				as := in["as"].(string)
				idx, hasIdx := in["index"].(string)
				base := len(m.locals)
				m.locals = append(m.locals, localVar{name: as})
				if hasIdx {
					m.locals = append(m.locals, localVar{name: idx})
				}
				for i, el := range arr {
					m.Charge(m.costs.LoopIter.At(int64(len(arr))))
					m.locals[base].v = el
					if hasIdx {
						m.locals[base+1].v = ratInt(i)
					}
					m.run(in["do"].([]any))
				}
				m.dropLocals(base)
			},
			Compile: func(c *Compiler, in map[string]any) func(*Machine) {
				over, body := c.Operand(in["in"]), c.Block(in["do"])
				as := in["as"].(string)
				idx, hasIdx := in["index"].(string)
				return func(m *Machine) {
					arr, ok := over(m).([]any)
					if !ok {
						execFail("TYPE_ERROR", "for: in must be an array")
					}
					each := m.costs.LoopIter.At(int64(len(arr)))
					base := len(m.locals)
					m.locals = append(m.locals, localVar{name: as})
					if hasIdx {
						m.locals = append(m.locals, localVar{name: idx})
					}
					for i, el := range arr {
						m.Charge(each)
						m.locals[base].v = el
						if hasIdx {
							m.locals[base+1].v = ratInt(i)
						}
						body(m)
					}
					m.dropLocals(base)
				}
			},
		},
		"assert": {
			Name: "assert", Req: []string{"that"}, Opt: []string{"msg"},
			Check: func(in map[string]any, c *Checker) {
				checkOperand(in["that"], c)
				if msg, has := in["msg"]; has {
					if _, ok := msg.(string); !ok {
						progFail("assert: msg must be a string")
					}
				}
			},
			Exec: func(m *Machine, in map[string]any) {
				if !boo(m.eval(in["that"])) {
					msg, _ := in["msg"].(string)
					execFail("ASSERTION_FAILED", "%s", msg)
				}
			},
			Compile: func(c *Compiler, in map[string]any) func(*Machine) {
				that := c.Operand(in["that"])
				msg, _ := in["msg"].(string)
				return func(m *Machine) {
					if !boo(that(m)) {
						execFail("ASSERTION_FAILED", "%s", msg)
					}
				}
			},
		},
		"halt": {
			Name:  "halt",
			Check: func(in map[string]any, c *Checker) {},
			Exec:  func(m *Machine, in map[string]any) { panic(haltSignal{}) },
			Compile: func(c *Compiler, in map[string]any) func(*Machine) {
				return func(*Machine) { panic(haltSignal{}) }
			},
		},
	}
}

// checkBlock statically checks a program block (an array of instructions)
// against the instruction table and locals carried in c. Every
// instruction shares the same generic "missing required field" / "unknown
// field" validation, driven by the looked-up InstructionDef's Req/Opt;
// only what's specific to one opcode lives in that opcode's own Check.
func checkBlock(b any, c *Checker) {
	arr, ok := b.([]any)
	if !ok {
		progFail("a block must be an array")
	}
	for _, raw := range arr {
		in, ok := raw.(map[string]any)
		if !ok {
			progFail("an instruction must be an object")
		}
		opn, ok := in["op"].(string)
		if !ok {
			progFail("instruction without a string op")
		}
		def, ok := c.Instructions[opn]
		if !ok {
			progFail("unknown op %q", opn)
		}
		allowed := map[string]bool{"op": true}
		for _, f := range def.Req {
			allowed[f] = true
			if _, has := in[f]; !has {
				progFail("%s: missing field %q", opn, f)
			}
		}
		for _, f := range def.Opt {
			allowed[f] = true
		}
		for k := range in {
			if !allowed[k] {
				progFail("%s: unknown field %q", opn, k)
			}
		}
		def.Check(in, c)
	}
}

// CheckProgram statically checks a program block against the given
// instruction table — CoreInstructions(), or CoreInstructions() plus a
// host's own entries — and returns a documented Err instead of panicking,
// for callers outside this package's own Run (a host's own Run needs this
// to validate a program that may contain its own extra instructions
// before ever executing it).
func CheckProgram(program any, instructions map[string]InstructionDef) (err *Err) {
	return CheckProgramHost(program, instructions, nil)
}

// CheckProgramHost is CheckProgram with the Checker's Host slot populated,
// so a registered instruction's Check function can consult host-specific
// state (Shaxon's "check" confirming its shape name is declared). Without
// this, no exported path could set Checker.Host at all.
func CheckProgramHost(program any, instructions map[string]InstructionDef, host any) (err *Err) {
	defer func() {
		if r := recover(); r != nil {
			e, ok := r.(*Err)
			if !ok {
				panic(r)
			}
			err = e
		}
	}()
	c := NewChecker(instructions)
	c.Host = host
	checkBlock(program, c)
	return nil
}
