// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the GNU General Public License, version 3.
// https://www.gnu.org/licenses/gpl-3.0.html
package build

// ---------------------------------------------------------------- instructions

// Instr is one instruction of a program block. It is a sealed interface;
// the constructors below (and the If/For chains) are the only way to make
// one.
type Instr interface{ instrNode() map[string]any }

type instr map[string]any

func (i instr) instrNode() map[string]any { return i }

func block(is []Instr) []any {
	out := make([]any, len(is))
	for k, i := range is {
		out[k] = i.instrNode()
	}
	return out
}

// Set writes value at target. Writing to Input, or to a loop-local path,
// is rejected when the package is assembled.
func Set(target Path, value Operand) Instr {
	return instr{"op": "set", "path": target.raw(), "value": value.operandNode()}
}

// Append adds value to the end of the array at target.
func Append(target Path, value Operand) Instr {
	return instr{"op": "append", "path": target.raw(), "value": value.operandNode()}
}

// Insert places value into the array at target, before position at.
func Insert(target Path, at, value Operand) Instr {
	return instr{"op": "insert", "path": target.raw(), "at": at.operandNode(), "value": value.operandNode()}
}

// Delete removes the member or element at target. Later array elements
// shift down.
func Delete(target Path) Instr {
	return instr{"op": "delete", "path": target.raw()}
}

// Halt stops execution normally; the output as built so far is returned.
func Halt() Instr { return instr{"op": "halt"} }

// AssertInstr is an assertion; add a failure message with Msg.
type AssertInstr struct {
	that Operand
	msg  string
	has  bool
}

// Assert fails execution unless that evaluates to true.
func Assert(that Operand) AssertInstr { return AssertInstr{that: that} }

// Msg sets the message reported when the assertion fails.
func (a AssertInstr) Msg(m string) AssertInstr { a.msg, a.has = m, true; return a }

func (a AssertInstr) instrNode() map[string]any {
	m := map[string]any{"op": "assert", "that": a.that.operandNode()}
	if a.has {
		m["msg"] = a.msg
	}
	return m
}

// IfCond is the first half of a conditional. It is not an Instr: it
// becomes one only when Then supplies the taken branch.
type IfCond struct{ cond Operand }

// If begins a conditional: jb.If(c).Then(...).Else(...).
func If(cond Operand) IfCond { return IfCond{cond} }

// Then supplies the branch taken when the condition is true.
func (c IfCond) Then(body ...Instr) IfInstr { return IfInstr{cond: c.cond, then: body} }

// IfInstr is a complete conditional; Else is optional.
type IfInstr struct {
	cond      Operand
	then, els []Instr
	hasElse   bool
}

// Else supplies the branch taken when the condition is false.
func (i IfInstr) Else(body ...Instr) IfInstr { i.els, i.hasElse = body, true; return i }

func (i IfInstr) instrNode() map[string]any {
	m := map[string]any{"op": "if", "cond": i.cond.operandNode(), "then": block(i.then)}
	if i.hasElse {
		m["else"] = block(i.els)
	}
	return m
}

// ForIn is the first stage of a loop: the collection. Not yet an Instr.
type ForIn struct{ in Operand }

// For begins a loop over an array: jb.For(xs).As(x).Do(...). The loop
// iterates a snapshot taken before the first pass.
func For(in Operand) ForIn { return ForIn{in} }

// As names the element binding.
func (f ForIn) As(v Loop) ForAs { return ForAs{in: f.in, as: v} }

// ForAs is a loop with its element bound; Index is optional, Do completes it.
type ForAs struct {
	in    Operand
	as    Loop
	index Loop
}

// Index also binds the zero-based position under the given name.
func (f ForAs) Index(v Loop) ForAs { f.index = v; return f }

// Do supplies the loop body and completes the instruction.
func (f ForAs) Do(body ...Instr) Instr {
	m := map[string]any{"op": "for", "in": f.in.operandNode(), "as": string(f.as), "do": block(body)}
	if f.index != "" {
		m["index"] = string(f.index)
	}
	return instr(m)
}
