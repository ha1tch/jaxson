// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the GNU General Public License, version 3.
// https://www.gnu.org/licenses/gpl-3.0.html
//
// jaxrun is a throwaway reference interpreter used to check the Jaxson v1
// core design against its golden fixtures. It is a specification check, not
// a product.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"
)

// ---------------------------------------------------------------- errors

// Err is a Jaxson failure with a category and an optional stable code.
type Err struct{ Cat, Code, Msg string }

func (e *Err) Error() string { return e.Cat + "/" + e.Code + ": " + e.Msg }

func fail(cat, code, format string, a ...any) {
	panic(&Err{cat, code, fmt.Sprintf(format, a...)})
}
func execFail(code, format string, a ...any) { fail("EXECUTION_ERROR", code, format, a...) }
func progFail(format string, a ...any)       { fail("PROGRAM_ERROR", "", format, a...) }

type haltSignal struct{}
type omitT struct{}

var omit = omitT{}

// ---------------------------------------------------------------- numbers

var ten = big.NewRat(10, 1)

const maxDigits = 38

// decParts returns the coefficient and the minimal scale of a decimal Rat.
func decParts(r *big.Rat) (*big.Int, int, bool) {
	n := new(big.Rat).Set(r)
	s := 0
	for !n.IsInt() {
		if s > 200 {
			return nil, 0, false
		}
		n.Mul(n, ten)
		s++
	}
	return new(big.Int).Set(n.Num()), s, true
}

func checkNum(r *big.Rat) *big.Rat {
	c, _, ok := decParts(r)
	if !ok || len(new(big.Int).Abs(c).String()) > maxDigits {
		execFail("NUMERIC_OVERFLOW", "number exceeds %d digits", maxDigits)
	}
	return r
}

func fmtNum(r *big.Rat) string {
	c, s, _ := decParts(r)
	neg := c.Sign() < 0
	d := new(big.Int).Abs(c).String()
	if s > 0 {
		for len(d) <= s {
			d = "0" + d
		}
		d = d[:len(d)-s] + "." + d[len(d)-s:]
	}
	if neg {
		d = "-" + d
	}
	return d
}

func roundRat(x *big.Rat, scale int, mode string) *big.Rat {
	p := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(scale)), nil)
	m := new(big.Rat).Mul(x, new(big.Rat).SetInt(p))
	neg := m.Sign() < 0
	a := new(big.Rat).Abs(m)
	fl := new(big.Int).Div(a.Num(), a.Denom())
	rem := new(big.Rat).Sub(a, new(big.Rat).SetInt(fl))
	half := big.NewRat(1, 2)
	inc := false
	switch mode {
	case "half_up":
		inc = rem.Cmp(half) >= 0
	case "half_even":
		c := rem.Cmp(half)
		inc = c > 0 || (c == 0 && fl.Bit(0) == 1)
	case "down":
		inc = false
	default:
		execFail("BAD_MODE", "unknown rounding mode %q", mode)
	}
	if inc {
		fl.Add(fl, big.NewInt(1))
	}
	if neg {
		fl.Neg(fl)
	}
	return new(big.Rat).SetFrac(fl, p)
}

// ---------------------------------------------------------------- values

func norm(v any) any {
	switch t := v.(type) {
	case json.Number:
		r, ok := new(big.Rat).SetString(t.String())
		if !ok {
			fail("PARSE_ERROR", "", "bad number %s", t)
		}
		return r
	case []any:
		for i := range t {
			t[i] = norm(t[i])
		}
		return t
	case map[string]any:
		for k, x := range t {
			t[k] = norm(x)
		}
		return t
	}
	return v
}

func clone(v any) any {
	switch t := v.(type) {
	case *big.Rat:
		return new(big.Rat).Set(t)
	case []any:
		o := make([]any, len(t))
		for i, x := range t {
			o[i] = clone(x)
		}
		return o
	case map[string]any:
		o := make(map[string]any, len(t))
		for k, x := range t {
			o[k] = clone(x)
		}
		return o
	}
	return v
}

func equal(a, b any) bool {
	switch x := a.(type) {
	case nil:
		return b == nil
	case bool:
		y, ok := b.(bool)
		return ok && x == y
	case string:
		y, ok := b.(string)
		return ok && x == y
	case *big.Rat:
		y, ok := b.(*big.Rat)
		return ok && x.Cmp(y) == 0
	case []any:
		y, ok := b.([]any)
		if !ok || len(x) != len(y) {
			return false
		}
		for i := range x {
			if !equal(x[i], y[i]) {
				return false
			}
		}
		return true
	case map[string]any:
		y, ok := b.(map[string]any)
		if !ok || len(x) != len(y) {
			return false
		}
		for k, xv := range x {
			yv, ok := y[k]
			if !ok || !equal(xv, yv) {
				return false
			}
		}
		return true
	}
	return false
}

func typeName(v any) string {
	switch v.(type) {
	case nil:
		return "null"
	case bool:
		return "boolean"
	case *big.Rat:
		return "number"
	case string:
		return "string"
	case []any:
		return "array"
	case map[string]any:
		return "object"
	}
	return "unknown"
}

func sortedKeys(m map[string]any) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	return ks
}

func show(v any) string {
	switch t := v.(type) {
	case nil:
		return "null"
	case bool, string:
		b, _ := json.Marshal(t)
		return string(b)
	case *big.Rat:
		return fmtNum(t)
	case []any:
		parts := make([]string, len(t))
		for i, x := range t {
			parts[i] = show(x)
		}
		return "[" + strings.Join(parts, ",") + "]"
	case map[string]any:
		var parts []string
		for _, k := range sortedKeys(t) {
			kb, _ := json.Marshal(k)
			parts = append(parts, string(kb)+":"+show(t[k]))
		}
		return "{" + strings.Join(parts, ",") + "}"
	}
	return "?"
}

// ---------------------------------------------------------------- static validation

var roots = map[string]bool{"input": true, "state": true, "output": true, "local": true}
var nameRe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
var numRe = regexp.MustCompile(`^-?(0|[1-9][0-9]*)(\.[0-9]+)?$`)

var arity = map[string][2]int{
	"add": {2, -1}, "sub": {2, -1}, "mul": {2, -1}, "neg": {1, 1}, "abs": {1, 1},
	"min": {1, -1}, "max": {1, -1}, "mod": {2, 2}, "div": {3, 4}, "round": {2, 3},
	"eq": {2, 2}, "ne": {2, 2}, "lt": {2, 2}, "le": {2, 2}, "gt": {2, 2}, "ge": {2, 2},
	"and": {1, -1}, "or": {1, -1}, "not": {1, 1}, "select": {3, 3},
	"concat": {1, -1}, "len": {1, 1}, "to_string": {1, 1}, "to_number": {1, 1},
	"list": {0, -1}, "get": {2, 2}, "has": {2, 2}, "get_or": {3, 3}, "keys": {1, 1}, "type_of": {1, 1},
}

type spec struct{ req, opt []string }

var instrs = map[string]spec{
	"set":    {[]string{"path", "value"}, nil},
	"delete": {[]string{"path"}, nil},
	"append": {[]string{"path", "value"}, nil},
	"insert": {[]string{"path", "at", "value"}, nil},
	"if":     {[]string{"cond", "then"}, []string{"else"}},
	"for":    {[]string{"in", "as", "do"}, []string{"index"}},
	"assert": {[]string{"that"}, []string{"msg"}},
	"halt":   {nil, nil},
}

func copyLocals(l map[string]bool) map[string]bool {
	o := make(map[string]bool, len(l)+2)
	for k, v := range l {
		o[k] = v
	}
	return o
}

func checkBlock(b any, locals map[string]bool) {
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
		sp, ok := instrs[opn]
		if !ok {
			progFail("unknown op %q", opn)
		}
		allowed := map[string]bool{"op": true}
		for _, f := range sp.req {
			allowed[f] = true
			if _, has := in[f]; !has {
				progFail("%s: missing field %q", opn, f)
			}
		}
		for _, f := range sp.opt {
			allowed[f] = true
		}
		for k := range in {
			if !allowed[k] {
				progFail("%s: unknown field %q", opn, k)
			}
		}
		switch opn {
		case "set", "append":
			checkPath(in["path"], locals, true)
			checkOperand(in["value"], locals)
		case "insert":
			checkPath(in["path"], locals, true)
			checkOperand(in["at"], locals)
			checkOperand(in["value"], locals)
		case "delete":
			checkPath(in["path"], locals, true)
			if len(in["path"].([]any)) < 2 {
				progFail("delete needs a path below a root")
			}
		case "if":
			checkOperand(in["cond"], locals)
			checkBlock(in["then"], locals)
			if e, has := in["else"]; has {
				checkBlock(e, locals)
			}
		case "for":
			checkOperand(in["in"], locals)
			as, ok := in["as"].(string)
			if !ok || !nameRe.MatchString(as) || locals[as] {
				progFail("for: bad or shadowed binding name")
			}
			inner := copyLocals(locals)
			inner[as] = true
			if ix, has := in["index"]; has {
				is, ok := ix.(string)
				if !ok || !nameRe.MatchString(is) || locals[is] || is == as {
					progFail("for: bad or shadowed index name")
				}
				inner[is] = true
			}
			checkBlock(in["do"], inner)
		case "assert":
			checkOperand(in["that"], locals)
			if msg, has := in["msg"]; has {
				if _, ok := msg.(string); !ok {
					progFail("assert: msg must be a string")
				}
			}
		}
	}
}

func checkPath(p any, locals map[string]bool, write bool) {
	arr, ok := p.([]any)
	if !ok || len(arr) == 0 {
		progFail("a path must be a non-empty array")
	}
	root, ok := arr[0].(string)
	if !ok || !roots[root] {
		progFail("bad path root")
	}
	if write && root != "state" && root != "output" {
		progFail("cannot write to root %q", root)
	}
	if root == "local" {
		if len(arr) < 2 {
			progFail("a local path needs a name")
		}
		name, ok := arr[1].(string)
		if !ok || !locals[name] {
			progFail("undeclared local")
		}
	}
	for _, s := range arr[1:] {
		switch t := s.(type) {
		case string:
		case *big.Rat:
			if !t.IsInt() || t.Sign() < 0 {
				progFail("a literal index must be a non-negative integer")
			}
		case map[string]any:
			isForm := false
			for k := range t {
				if strings.HasPrefix(k, "$") {
					isForm = true
				}
			}
			if !isForm {
				progFail("a path segment must be a string, an index or a form")
			}
			checkOperand(t, locals)
		default:
			progFail("bad path segment")
		}
	}
}

func containsForm(x any) bool {
	switch t := x.(type) {
	case map[string]any:
		for k, v := range t {
			if strings.HasPrefix(k, "$") || containsForm(v) {
				return true
			}
		}
	case []any:
		for _, v := range t {
			if containsForm(v) {
				return true
			}
		}
	}
	return false
}

func checkOperand(x any, locals map[string]bool) {
	if mp, ok := x.(map[string]any); ok {
		for k := range mp {
			if strings.HasPrefix(k, "$") {
				checkForm(mp, locals)
				return
			}
		}
	}
	if containsForm(x) {
		progFail("a literal contains a $-form; use $tpl or $lit")
	}
}

func checkForm(mp map[string]any, locals map[string]bool) {
	if len(mp) != 1 {
		progFail("a $-form must have exactly one key")
	}
	for k, v := range mp {
		switch k {
		case "$path":
			checkPath(v, locals, false)
		case "$lit":
		case "$compute":
			checkCompute(v, locals)
		case "$tpl":
			checkTpl(v, locals, true)
		case "$opt":
			progFail("$opt is only valid inside $tpl")
		case "$v":
			progFail("$v is only valid inside a compute expr")
		default:
			progFail("unknown form %q", k)
		}
	}
}

func checkTpl(v any, locals map[string]bool, top bool) {
	switch t := v.(type) {
	case map[string]any:
		if len(t) == 1 {
			for k, a := range t {
				if k == "$opt" {
					if top {
						progFail("a $tpl root cannot be $opt")
					}
					checkPath(a, locals, false)
					return
				}
				if strings.HasPrefix(k, "$") {
					checkForm(t, locals)
					return
				}
			}
		}
		for k, a := range t {
			if strings.HasPrefix(k, "$") {
				progFail("a $-key must be the only key of its object")
			}
			checkTpl(a, locals, false)
		}
	case []any:
		for _, a := range t {
			checkTpl(a, locals, false)
		}
	}
}

func checkCompute(v any, locals map[string]bool) {
	c, ok := v.(map[string]any)
	if !ok {
		progFail("$compute must be an object")
	}
	for k := range c {
		if k != "with" && k != "expr" {
			progFail("$compute: unknown key %q", k)
		}
	}
	if _, ok := c["expr"]; !ok {
		progFail("$compute needs an expr")
	}
	names := map[string]bool{}
	if w, has := c["with"]; has {
		wm, ok := w.(map[string]any)
		if !ok {
			progFail("$compute: with must be an object")
		}
		for n, o := range wm {
			if !nameRe.MatchString(n) {
				progFail("$compute: bad binding name %q", n)
			}
			names[n] = true
			checkOperand(o, locals)
		}
	}
	checkExpr(c["expr"], names)
}

func checkExpr(e any, names map[string]bool) {
	switch t := e.(type) {
	case nil, bool, string, *big.Rat:
		return
	case map[string]any:
		n, ok := t["$v"].(string)
		if len(t) != 1 || !ok || !names[n] {
			progFail("bad $v reference in expr")
		}
	case []any:
		if len(t) == 0 {
			progFail("an empty application in expr")
		}
		op, ok := t[0].(string)
		ar, known := arity[op]
		if !ok || !known {
			progFail("unknown compute operation %v", t[0])
		}
		n := len(t) - 1
		if n < ar[0] || (ar[1] >= 0 && n > ar[1]) {
			progFail("wrong arity for %s", op)
		}
		for _, a := range t[1:] {
			checkExpr(a, names)
		}
	}
}

// ---------------------------------------------------------------- contract schemas

var schemaKeys = map[string][]string{
	"any": {}, "null": {}, "boolean": {},
	"number": {"int", "min", "max"}, "string": {"minLen", "maxLen", "enum"},
	"array": {"items", "minItems", "maxItems"}, "object": {"fields", "required", "extra"},
}

func checkSchema(s any) {
	m, ok := s.(map[string]any)
	if !ok {
		fail("SCHEMA_ERROR", "", "a schema must be an object")
	}
	if a, has := m["anyOf"]; has {
		for k := range m {
			if k != "anyOf" && k != "nullable" {
				fail("SCHEMA_ERROR", "", "anyOf schema has extra keyword %q", k)
			}
		}
		arr, ok := a.([]any)
		if !ok || len(arr) == 0 {
			fail("SCHEMA_ERROR", "", "anyOf needs a non-empty array")
		}
		for _, sub := range arr {
			checkSchema(sub)
		}
		return
	}
	t, ok := m["type"].(string)
	keys, known := schemaKeys[t]
	if !ok || !known {
		fail("SCHEMA_ERROR", "", "unknown or missing schema type")
	}
	allowed := map[string]bool{"type": true, "nullable": true}
	for _, k := range keys {
		allowed[k] = true
	}
	for k := range m {
		if !allowed[k] {
			fail("SCHEMA_ERROR", "", "keyword %q is not valid for type %s", k, t)
		}
	}
	switch t {
	case "array":
		if it, has := m["items"]; has {
			checkSchema(it)
		}
	case "object":
		fields, _ := m["fields"].(map[string]any)
		for _, f := range fields {
			checkSchema(f)
		}
		if req, has := m["required"]; has {
			arr, ok := req.([]any)
			if !ok {
				fail("SCHEMA_ERROR", "", "required must be an array")
			}
			for _, n := range arr {
				ns, ok := n.(string)
				if _, declared := fields[ns]; !ok || !declared {
					fail("SCHEMA_ERROR", "", "required names an undeclared field")
				}
			}
		}
		if ex, has := m["extra"]; has && ex != "reject" && ex != "allow" {
			fail("SCHEMA_ERROR", "", "extra must be reject or allow")
		}
	}
}

func ratOf(v any) *big.Rat { r, _ := v.(*big.Rat); return r }

// validate returns "" when v satisfies s, otherwise the first failure.
func validate(s, v any, path string) string {
	m := s.(map[string]any)
	if nl, _ := m["nullable"].(bool); nl && v == nil {
		return ""
	}
	if a, has := m["anyOf"]; has {
		for _, sub := range a.([]any) {
			if validate(sub, v, path) == "" {
				return ""
			}
		}
		return path + ": matches no anyOf alternative"
	}
	t := m["type"].(string)
	if t == "any" {
		return ""
	}
	if got := typeName(v); got != t {
		return fmt.Sprintf("%s: expected %s, got %s", path, t, got)
	}
	switch t {
	case "number":
		r := v.(*big.Rat)
		if i, _ := m["int"].(bool); i && !r.IsInt() {
			return path + ": expected an integer"
		}
		if lo := ratOf(m["min"]); lo != nil && r.Cmp(lo) < 0 {
			return path + ": below minimum"
		}
		if hi := ratOf(m["max"]); hi != nil && r.Cmp(hi) > 0 {
			return path + ": above maximum"
		}
	case "string":
		str := v.(string)
		n := int64(utf8.RuneCountInString(str))
		if lo := ratOf(m["minLen"]); lo != nil && big.NewRat(n, 1).Cmp(lo) < 0 {
			return path + ": too short"
		}
		if hi := ratOf(m["maxLen"]); hi != nil && big.NewRat(n, 1).Cmp(hi) > 0 {
			return path + ": too long"
		}
		if en, has := m["enum"].([]any); has {
			found := false
			for _, e := range en {
				if e == str {
					found = true
				}
			}
			if !found {
				return path + ": not in enum"
			}
		}
	case "array":
		arr := v.([]any)
		n := big.NewRat(int64(len(arr)), 1)
		if lo := ratOf(m["minItems"]); lo != nil && n.Cmp(lo) < 0 {
			return path + ": too few items"
		}
		if hi := ratOf(m["maxItems"]); hi != nil && n.Cmp(hi) > 0 {
			return path + ": too many items"
		}
		if it, has := m["items"]; has {
			for i, e := range arr {
				if msg := validate(it, e, fmt.Sprintf("%s[%d]", path, i)); msg != "" {
					return msg
				}
			}
		}
	case "object":
		obj := v.(map[string]any)
		fields, _ := m["fields"].(map[string]any)
		if req, has := m["required"].([]any); has {
			for _, n := range req {
				if _, present := obj[n.(string)]; !present {
					return fmt.Sprintf("%s: missing required %q", path, n)
				}
			}
		}
		for _, k := range sortedKeys(obj) {
			if fs, declared := fields[k]; declared {
				if msg := validate(fs, obj[k], path+"."+k); msg != "" {
					return msg
				}
			} else if m["extra"] != "allow" {
				return fmt.Sprintf("%s: unexpected member %q", path, k)
			}
		}
	}
	return ""
}

// ---------------------------------------------------------------- machine

type machine struct {
	input, state, output any
	locals               map[string]any
	steps, limit         int
}

func (m *machine) step() {
	m.steps++
	if m.steps > m.limit {
		fail("RESOURCE_ERROR", "STEPS", "step limit %d exceeded", m.limit)
	}
}

func (m *machine) toIndex(r *big.Rat) int {
	if !r.IsInt() || r.Sign() < 0 || r.Num().BitLen() > 31 {
		execFail("BAD_INDEX", "invalid index %s", fmtNum(r))
	}
	return int(r.Num().Int64())
}

func (m *machine) segs(p []any) (string, []any) {
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
			execFail("TYPE_ERROR", "a path segment must be a string or an integer, got %s", typeName(v))
		}
	}
	return root, out
}

func (m *machine) walk(root string, segs []any) (any, *Err) {
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
				return nil, &Err{"EXECUTION_ERROR", "TYPE_ERROR", fmt.Sprintf("cannot read member %q of %s", k, typeName(cur))}
			}
			v, ok := mp[k]
			if !ok {
				return nil, &Err{"EXECUTION_ERROR", "MISSING_PATH", fmt.Sprintf("member %q not found", k)}
			}
			cur = v
		case int:
			ar, ok := cur.([]any)
			if !ok {
				return nil, &Err{"EXECUTION_ERROR", "TYPE_ERROR", fmt.Sprintf("cannot index %s", typeName(cur))}
			}
			if k >= len(ar) {
				return nil, &Err{"EXECUTION_ERROR", "MISSING_PATH", fmt.Sprintf("index %d out of range", k)}
			}
			cur = ar[k]
		}
	}
	return cur, nil
}

func (m *machine) getAt(root string, segs []any) any {
	v, e := m.walk(root, segs)
	if e != nil {
		panic(e)
	}
	return v
}

func (m *machine) putAt(root string, segs []any, v any) {
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
			execFail("TYPE_ERROR", "cannot set member %q on %s", k, typeName(par))
		}
		mp[k] = v
	case int:
		ar, ok := par.([]any)
		if !ok {
			execFail("TYPE_ERROR", "cannot set an index on %s", typeName(par))
		}
		if k >= len(ar) {
			execFail("BAD_INDEX", "index %d out of range", k)
		}
		ar[k] = v
	}
}

func (m *machine) read(p []any) any {
	r, s := m.segs(p)
	return clone(m.getAt(r, s))
}

func (m *machine) eval(x any) any {
	if mp, ok := x.(map[string]any); ok && len(mp) == 1 {
		for k, v := range mp {
			switch k {
			case "$path":
				return m.read(v.([]any))
			case "$lit":
				return clone(v)
			case "$compute":
				return clone(m.compute(v.(map[string]any)))
			case "$tpl":
				return m.tpl(v)
			}
		}
	}
	return clone(x)
}

func (m *machine) tpl(v any) any {
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
					return clone(val)
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
	return clone(v)
}

func (m *machine) compute(c map[string]any) any {
	env := map[string]any{}
	if w, ok := c["with"].(map[string]any); ok {
		for _, n := range sortedKeys(w) {
			env[n] = m.eval(w[n])
		}
	}
	return m.expr(c["expr"], env)
}

func (m *machine) expr(e any, env map[string]any) any {
	switch t := e.(type) {
	case map[string]any:
		return env[t["$v"].(string)]
	case []any:
		m.step()
		return m.apply(t[0].(string), t[1:], env)
	}
	return e
}

func num(x any) *big.Rat {
	r, ok := x.(*big.Rat)
	if !ok {
		execFail("TYPE_ERROR", "expected number, got %s", typeName(x))
	}
	return r
}
func str(x any) string {
	s, ok := x.(string)
	if !ok {
		execFail("TYPE_ERROR", "expected string, got %s", typeName(x))
	}
	return s
}
func boo(x any) bool {
	b, ok := x.(bool)
	if !ok {
		execFail("TYPE_ERROR", "expected boolean, got %s", typeName(x))
	}
	return b
}
func integer(x any) *big.Int {
	r := num(x)
	if !r.IsInt() {
		execFail("TYPE_ERROR", "expected integer, got %s", fmtNum(r))
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
func order(a, b any) int {
	switch x := a.(type) {
	case *big.Rat:
		if y, ok := b.(*big.Rat); ok {
			return x.Cmp(y)
		}
	case string:
		if y, ok := b.(string); ok {
			return strings.Compare(x, y)
		}
	}
	execFail("TYPE_ERROR", "cannot order %s and %s", typeName(a), typeName(b))
	return 0
}

func (m *machine) apply(op string, a []any, env map[string]any) any {
	switch op {
	case "and":
		for _, x := range a {
			if !boo(m.expr(x, env)) {
				return false
			}
		}
		return true
	case "or":
		for _, x := range a {
			if boo(m.expr(x, env)) {
				return true
			}
		}
		return false
	case "select":
		if boo(m.expr(a[0], env)) {
			return m.expr(a[1], env)
		}
		return m.expr(a[2], env)
	}
	v := make([]any, len(a))
	for i, x := range a {
		v[i] = m.expr(x, env)
	}
	switch op {
	case "add":
		s := new(big.Rat)
		for _, x := range v {
			s.Add(s, num(x))
		}
		return checkNum(s)
	case "sub":
		s := new(big.Rat).Set(num(v[0]))
		for _, x := range v[1:] {
			s.Sub(s, num(x))
		}
		return checkNum(s)
	case "mul":
		s := big.NewRat(1, 1)
		for _, x := range v {
			s.Mul(s, num(x))
		}
		return checkNum(s)
	case "neg":
		return new(big.Rat).Neg(num(v[0]))
	case "abs":
		return new(big.Rat).Abs(num(v[0]))
	case "min", "max":
		best := num(v[0])
		for _, x := range v[1:] {
			c := num(x).Cmp(best)
			if (op == "min" && c < 0) || (op == "max" && c > 0) {
				best = num(x)
			}
		}
		return new(big.Rat).Set(best)
	case "mod":
		x, y := integer(v[0]), integer(v[1])
		if y.Sign() == 0 {
			execFail("DIV_ZERO", "modulo by zero")
		}
		return new(big.Rat).SetInt(new(big.Int).Rem(x, y))
	case "div":
		x, y := num(v[0]), num(v[1])
		if y.Sign() == 0 {
			execFail("DIV_ZERO", "division by zero")
		}
		mode := "half_even"
		if len(v) == 4 {
			mode = str(v[3])
		}
		return checkNum(roundRat(new(big.Rat).Quo(x, y), scaleOf(v[2]), mode))
	case "round":
		mode := "half_even"
		if len(v) == 3 {
			mode = str(v[2])
		}
		return checkNum(roundRat(num(v[0]), scaleOf(v[1]), mode))
	case "eq":
		return equal(v[0], v[1])
	case "ne":
		return !equal(v[0], v[1])
	case "lt":
		return order(v[0], v[1]) < 0
	case "le":
		return order(v[0], v[1]) <= 0
	case "gt":
		return order(v[0], v[1]) > 0
	case "ge":
		return order(v[0], v[1]) >= 0
	case "not":
		return !boo(v[0])
	case "concat":
		var b strings.Builder
		for _, x := range v {
			b.WriteString(str(x))
		}
		return b.String()
	case "len":
		switch t := v[0].(type) {
		case string:
			return big.NewRat(int64(utf8.RuneCountInString(t)), 1)
		case []any:
			return big.NewRat(int64(len(t)), 1)
		case map[string]any:
			return big.NewRat(int64(len(t)), 1)
		}
		execFail("TYPE_ERROR", "len of %s", typeName(v[0]))
	case "to_string":
		return fmtNum(num(v[0]))
	case "to_number":
		s := str(v[0])
		if !numRe.MatchString(s) {
			execFail("BAD_NUMBER", "not a JSON number: %q", s)
		}
		r, _ := new(big.Rat).SetString(s)
		return checkNum(r)
	case "list":
		return v
	case "keys":
		mp, ok := v[0].(map[string]any)
		if !ok {
			execFail("TYPE_ERROR", "keys of %s", typeName(v[0]))
		}
		out := []any{}
		for _, k := range sortedKeys(mp) {
			out = append(out, k)
		}
		return out
	case "type_of":
		return typeName(v[0])
	case "has", "get", "get_or":
		val, found := lookup(v[0], v[1])
		if op == "has" {
			return found
		}
		if found {
			return val
		}
		if op == "get_or" {
			return v[2]
		}
		execFail("MISSING_PATH", "get: key not found")
	}
	panic("unreachable op " + op)
}

func lookup(c, k any) (any, bool) {
	switch t := c.(type) {
	case map[string]any:
		v, ok := t[str(k)]
		return v, ok
	case []any:
		i := new(machine).toIndex(num(k))
		if i < len(t) {
			return t[i], true
		}
		return nil, false
	}
	execFail("TYPE_ERROR", "cannot look up in %s", typeName(c))
	return nil, false
}

func (m *machine) run(block []any) {
	for _, raw := range block {
		in := raw.(map[string]any)
		m.step()
		switch in["op"].(string) {
		case "set":
			val := m.eval(in["value"])
			r, s := m.segs(in["path"].([]any))
			m.putAt(r, s, val)
		case "append":
			val := m.eval(in["value"])
			r, s := m.segs(in["path"].([]any))
			ar, ok := m.getAt(r, s).([]any)
			if !ok {
				execFail("TYPE_ERROR", "append target is not an array")
			}
			m.putAt(r, s, append(ar[:len(ar):len(ar)], val))
		case "insert":
			val := m.eval(in["value"])
			at := m.toIndex(num(m.eval(in["at"])))
			r, s := m.segs(in["path"].([]any))
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
		case "delete":
			r, s := m.segs(in["path"].([]any))
			par := m.getAt(r, s[:len(s)-1])
			switch k := s[len(s)-1].(type) {
			case string:
				mp, ok := par.(map[string]any)
				if !ok {
					execFail("TYPE_ERROR", "delete member of %s", typeName(par))
				}
				if _, has := mp[k]; !has {
					execFail("MISSING_PATH", "member %q not found", k)
				}
				delete(mp, k)
			case int:
				ar, ok := par.([]any)
				if !ok {
					execFail("TYPE_ERROR", "delete index of %s", typeName(par))
				}
				if k >= len(ar) {
					execFail("BAD_INDEX", "index %d out of range", k)
				}
				out := make([]any, 0, len(ar)-1)
				out = append(out, ar[:k]...)
				out = append(out, ar[k+1:]...)
				m.putAt(r, s[:len(s)-1], out)
			}
		case "if":
			if boo(m.eval(in["cond"])) {
				m.run(in["then"].([]any))
			} else if e, has := in["else"]; has {
				m.run(e.([]any))
			}
		case "for":
			arr, ok := m.eval(in["in"]).([]any)
			if !ok {
				execFail("TYPE_ERROR", "for: in must be an array")
			}
			as := in["as"].(string)
			idx, hasIdx := in["index"].(string)
			for i, el := range arr {
				m.step()
				m.locals[as] = el
				if hasIdx {
					m.locals[idx] = new(big.Rat).SetInt64(int64(i))
				}
				m.run(in["do"].([]any))
			}
			delete(m.locals, as)
			if hasIdx {
				delete(m.locals, idx)
			}
		case "assert":
			if !boo(m.eval(in["that"])) {
				msg, _ := in["msg"].(string)
				execFail("ASSERTION_FAILED", "%s", msg)
			}
		case "halt":
			panic(haltSignal{})
		}
	}
}

// Run executes a package and returns either its output or a failure.
func Run(p map[string]any) (out any, err *Err) {
	defer func() {
		if r := recover(); r != nil {
			e, ok := r.(*Err)
			if !ok {
				panic(r)
			}
			out, err = nil, e
		}
	}()
	if v, _ := p["jaxson"].(string); v != "1.0" {
		fail("VERSION_ERROR", "", "unsupported or missing jaxson version")
	}
	checkSchema(p["inputSchema"])
	checkSchema(p["outputSchema"])
	checkBlock(p["program"], map[string]bool{})
	limit := 100000
	if l, ok := p["limits"].(map[string]any); ok {
		for k := range l {
			if k != "steps" {
				fail("VERSION_ERROR", "", "unsupported limit %q", k)
			}
		}
		if s, ok := l["steps"].(*big.Rat); ok && s.IsInt() && s.Sign() > 0 {
			limit = int(s.Num().Int64())
		} else if _, has := l["steps"]; has {
			fail("VERSION_ERROR", "", "steps must be a positive integer")
		}
	}
	if msg := validate(p["inputSchema"], p["input"], "$"); msg != "" {
		fail("INPUT_ERROR", "", "%s", msg)
	}
	m := &machine{input: clone(p["input"]), state: map[string]any{}, locals: map[string]any{}, limit: limit}
	func() {
		defer func() {
			if r := recover(); r != nil {
				if _, ok := r.(haltSignal); !ok {
					panic(r)
				}
			}
		}()
		m.run(p["program"].([]any))
	}()
	if msg := validate(p["outputSchema"], m.output, "$"); msg != "" {
		fail("OUTPUT_ERROR", "", "%s", msg)
	}
	return m.output, nil
}

func check(exp map[string]any, out any, e *Err) (bool, string) {
	if want, has := exp["error"].(map[string]any); has {
		if e == nil {
			return false, "expected an error, got output " + show(out)
		}
		if e.Cat != want["category"] {
			return false, "wrong category: " + e.Error()
		}
		if c, has := want["code"]; has && c != e.Code {
			return false, "wrong code: " + e.Error()
		}
		return true, "-> " + e.Msg
	}
	if e != nil {
		return false, "unexpected error: " + e.Error()
	}
	if !equal(exp["output"], out) {
		return false, "output " + show(out) + " != expected " + show(exp["output"])
	}
	return true, ""
}

func main() {
	path := "fixtures.json"
	if len(os.Args) > 1 {
		path = os.Args[1]
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		fmt.Println(err)
		os.Exit(2)
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var cases []any
	if err := dec.Decode(&cases); err != nil {
		fmt.Println("fixtures are not valid JSON:", err)
		os.Exit(2)
	}
	norm(cases)
	pass, bad := 0, 0
	for _, c := range cases {
		cm := c.(map[string]any)
		out, e := Run(cm)
		ok, detail := check(cm["expect"].(map[string]any), out, e)
		if ok {
			pass++
			fmt.Printf("PASS  %-46s %s\n", cm["name"], detail)
		} else {
			bad++
			fmt.Printf("FAIL  %s: %s\n", cm["name"], detail)
		}
	}
	fmt.Printf("\n%d passed, %d failed\n", pass, bad)
	if bad > 0 {
		os.Exit(1)
	}
}
