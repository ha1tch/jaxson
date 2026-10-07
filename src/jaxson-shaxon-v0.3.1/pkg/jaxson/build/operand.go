// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the GNU General Public License, version 3.
// https://www.gnu.org/licenses/gpl-3.0.html
package build

import (
	"encoding/json"
)

// Operand is a value-producing form: a constant, a path, a template, a
// compute island, or raw/escaped data. It is a sealed interface; only this
// package implements it.
type Operand interface{ operandNode() any }

// Const is a scalar constant. It is both an Operand and an Expr, so
// Int(6) works on either side.
type Const struct{ v any }

func (c Const) operandNode() any { return c.v }
func (c Const) exprNode() any    { return c.v }

// Int is an integer constant.
func Int(n int) Const { return Const{n} }

// Dec is an exact decimal constant, written as text so no binary
// floating-point rounding can occur: Dec("0.10").
func Dec(s string) Const { return Const{json.Number(s)} }

// Str is a string constant.
func Str(s string) Const { return Const{s} }

// Bool is a boolean constant.
func Bool(b bool) Const { return Const{b} }

// Null is the null constant.
func Null() Const { return Const{nil} }

// Data is a raw JSON value used as a literal operand: a slice, a map, a
// string. It must not contain a $-form; Lit is the escape hatch for data
// that does. Package assembly rejects a Data that contains one.
func Data(v any) Operand { return data{v} }

type data struct{ v any }

func (d data) operandNode() any { return d.v }

// Lit is the $lit form: a literal taken verbatim, never interpreted,
// even if it looks like a form.
func Lit(v any) Operand { return lit{v} }

type lit struct{ v any }

func (l lit) operandNode() any { return map[string]any{"$lit": l.v} }

// ---------------------------------------------------------------- paths

// Path addresses a location under one of the roots input, state, output
// or local. It is an Operand (reading the value there) and the target of
// Set, Append, Insert and Delete. Writing to input or to a loop-local
// path is rejected when the package is assembled.
type Path struct {
	root string
	segs []any // string | int | Operand
}

// Input is a path under the read-only input root.
func Input(keys ...string) Path { return newPath("input", keys) }

// State is a path under the working-state root.
func State(keys ...string) Path { return newPath("state", keys) }

// Output is a path under the output root; Output() alone is the whole output.
func Output(keys ...string) Path { return newPath("output", keys) }

func newPath(root string, keys []string) Path {
	p := Path{root: root}
	for _, k := range keys {
		p.segs = append(p.segs, k)
	}
	return p
}

func (p Path) with(seg any) Path {
	n := Path{root: p.root, segs: make([]any, len(p.segs), len(p.segs)+1)}
	copy(n.segs, p.segs)
	n.segs = append(n.segs, seg)
	return n
}

// Key descends into an object member.
func (p Path) Key(k string) Path { return p.with(k) }

// Idx descends into an array element by constant index.
func (p Path) Idx(i int) Path { return p.with(i) }

// At descends by a segment computed at run time: a string key or a
// non-negative integer, produced by any Operand (Calc, another Path ...).
func (p Path) At(seg Operand) Path { return p.with(seg) }

func (p Path) raw() []any {
	out := make([]any, 0, len(p.segs)+1)
	out = append(out, p.root)
	for _, s := range p.segs {
		if o, ok := s.(Operand); ok {
			out = append(out, o.operandNode())
		} else {
			out = append(out, s)
		}
	}
	return out
}

func (p Path) operandNode() any { return map[string]any{"$path": p.raw()} }

// Loop names a loop-local binding: the element (As) or the position
// (Index) of a For. As an Operand it reads the binding; Key, Idx and At
// descend into it.
type Loop string

func (l Loop) operandNode() any { return l.path().operandNode() }
func (l Loop) path() Path       { return Path{root: "local", segs: []any{string(l)}} }

// Key descends into an object member of the bound value.
func (l Loop) Key(k string) Path { return l.path().Key(k) }

// Idx descends into an array element of the bound value.
func (l Loop) Idx(i int) Path { return l.path().Idx(i) }

// At descends by a run-time segment.
func (l Loop) At(seg Operand) Path { return l.path().At(seg) }

// ---------------------------------------------------------------- templates

// Template is the $tpl form: an object or array assembled from operands.
type Template struct {
	fields map[string]any // string -> Operand, or optField
	items  []Operand
	isList bool
}

type optField struct{ p Path }

// Tpl starts an object template. Chain Set and Opt.
func Tpl() Template { return Template{fields: map[string]any{}} }

// TplList starts an array template from its elements.
func TplList(items ...Operand) Template {
	return Template{isList: true, items: append([]Operand(nil), items...)}
}

// Set adds a member. A nested Template is embedded directly, not wrapped
// in a second $tpl.
func (t Template) Set(name string, v Operand) Template {
	n := t.clone()
	n.fields[name] = v
	return n
}

// Opt adds a member that is omitted from the result when the path is
// absent (a null there is kept; only absence drops the member). The
// runtime's $opt takes a path, not an arbitrary operand, and so does this.
func (t Template) Opt(name string, p Path) Template {
	n := t.clone()
	n.fields[name] = optField{p}
	return n
}

func (t Template) clone() Template {
	n := Template{isList: t.isList, items: t.items, fields: make(map[string]any, len(t.fields)+1)}
	for k, v := range t.fields {
		n.fields[k] = v
	}
	return n
}

func (t Template) operandNode() any { return map[string]any{"$tpl": t.inner()} }

func (t Template) inner() any {
	if t.isList {
		out := make([]any, len(t.items))
		for i, o := range t.items {
			out[i] = tplNode(o)
		}
		return out
	}
	out := make(map[string]any, len(t.fields))
	for k, v := range t.fields {
		if f, ok := v.(optField); ok {
			out[k] = map[string]any{"$opt": f.p.raw()}
		} else {
			out[k] = tplNode(v.(Operand))
		}
	}
	return out
}

func tplNode(o Operand) any {
	if t, ok := o.(Template); ok {
		return t.inner()
	}
	return o.operandNode()
}
