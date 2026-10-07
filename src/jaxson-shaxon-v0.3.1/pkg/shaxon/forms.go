// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the GNU General Public License, version 3.
// https://www.gnu.org/licenses/gpl-3.0.html
package shaxon

// Phase 4.4 (plan section 7): the path-expression operand forms of core
// section 6 — `$altPath`, `$inverse`, `$path*` and `$path+` — registered
// into jaxson's operand-form hook (jaxson/forms.go).
//
// This is an extension point beyond the ones plan section 4 listed ("and no
// others"): the core operand checker and evaluator knew four forms and had
// no way to be told about more. The hook is small and general (a form is a
// name plus a static check and an evaluation), and nothing in the core
// changes for a program that registers none.
//
// ====================================================================
// Decisions where the spec is silent or ambiguous (F1..F7)
// ====================================================================
//
//	F1  Values. In an operand position:
//	      $altPath   evaluates to the VALUE at the first alternative that
//	                 resolves, as `$path` does;
//	      $inverse   evaluates to an array of PATHS (core section 6, via the
//	                 v0.3.1 proposal 1.3), each an array of segments with the
//	                 root first, ["state", "items", 3];
//	      $path*/$path+  evaluate to an array of the PATHS in the closure.
//	    A path is a value like any other, so a `check` can compare two such
//	    lists (the cross-check section 4b describes) or count them; nothing
//	    in the operator table dereferences a path, and none is added.
//	F2  Not nestable. As the limitations document says, none of the four
//	    forms can contain another: the key of `$inverse` and the
//	    alternatives of `$altPath` may not themselves be one of these forms.
//	F3  Closures take their base from `from`, or, when it is omitted, from
//	    the ambient position: the focus node's path inside a shape `check`,
//	    the element's path inside an index `key`. The Machine knows only
//	    values, so the Evaluator and IndexSet publish that path through a
//	    small Ambient while they evaluate. With no `from` and no ambient
//	    position (a core `set`, say) the form is a static error where
//	    `local.focus` and `local.item` are not in scope, and a PROGRAM_ERROR
//	    at run time where a loop merely reuses the name `item`.
//	F4  A closure that reaches its maxDepth horizon with the chain still
//	    going, in an operand position, raises EXECUTION_ERROR/
//	    SHAX_PATH_DEPTH_EXCEEDED. As a target it is a reportable finding
//	    (paths.go C3); an operand has no report to attach to.
//	F5  The multi-key closure forms are not usable directly inside `$tpl`
//	    (whose own rule is that a $-key is its object's only key). Put them
//	    in a `with` binding or a `value` operand.
//	F6  `$inverse` names are checked once everything is parsed
//	    (checkInverseNames), since a `check` in a computes entry may be
//	    parsed before the relation it names. An index named by `$inverse`
//	    must be `multi`.
//	F7  An index whose key (transitively) asks `$inverse` of itself is a
//	    SHAPE_ERROR at build time rather than unbounded recursion.

import (
	"math/big"

	"github.com/ha1tch/jaxson/pkg/jaxson"
)

// Ambient is the path of the node being judged or indexed, published for
// the closure forms (F3). The zero value has no position.
type Ambient struct {
	path []any
	// While an index builds, the position is "element at of src". It is
	// only turned into a path if something asks for it (a closure form in
	// the key does; most keys never do), instead of one slice per element.
	src  []any
	at   int
	lazy bool
}

// Path returns the ambient position, or nil if there is none.
func (a *Ambient) Path() []any {
	if a.lazy {
		a.path, a.lazy, a.src = appendPath(a.src, a.at), false, nil
	}
	return a.path
}

func (a *Ambient) with(p []any, fn func()) {
	if a == nil {
		fn()
		return
	}
	old := *a
	*a = Ambient{path: p}
	defer func() { *a = old }()
	fn()
}

// withElem is with(appendPath(src, at), fn) without building the path unless
// it is read. A nil src means no position.
func (a *Ambient) withElem(src []any, at int, fn func()) {
	if a == nil {
		fn()
		return
	}
	old := *a
	if src == nil {
		*a = Ambient{}
	} else {
		*a = Ambient{src: src, at: at, lazy: true}
	}
	defer func() { *a = old }()
	fn()
}

var closureForms = []string{"$path*", "$path+"}

func isShaxonForm(x any) bool {
	mp, ok := x.(map[string]any)
	if !ok {
		return false
	}
	for _, k := range []string{"$altPath", "$inverse", "$path*", "$path+"} {
		if _, has := mp[k]; has {
			return true
		}
	}
	return false
}

func localNames(c *jaxson.Checker) []string {
	out := make([]string, 0, len(c.Locals))
	for n := range c.Locals {
		out = append(out, n)
	}
	return out
}

// StaticForms returns the four forms with their static checks only, for
// parse-time operand checking.
func StaticForms() map[string]jaxson.FormDef {
	return map[string]jaxson.FormDef{
		"$altPath": {Check: checkAltPath},
		"$inverse": {Check: checkInverse},
		"$path*":   {Check: func(f map[string]any, c *jaxson.Checker) { checkClosureForm("$path*", f, c) }},
		"$path+":   {Check: func(f map[string]any, c *jaxson.Checker) { checkClosureForm("$path+", f, c) }},
	}
}

func checkAltPath(form map[string]any, c *jaxson.Checker) {
	if len(form) != 1 {
		failLoad("$altPath takes only its list of alternatives")
	}
	alts, ok := form["$altPath"].([]any)
	if !ok || len(alts) == 0 {
		failLoad("$altPath needs a non-empty array of path alternatives")
	}
	for i, alt := range alts {
		if isShaxonForm(alt) {
			failLoad("$altPath[%d]: path-expression forms do not nest (F2)", i)
		}
		if e := jaxson.CheckPathWith(StaticForms(), alt, localNames(c)...); e != nil {
			failLoad("$altPath[%d]: %s", i, e.Msg)
		}
	}
}

func checkInverse(form map[string]any, c *jaxson.Checker) {
	if len(form) != 1 {
		failLoad("$inverse takes only its index/relation and key")
	}
	spec, ok := form["$inverse"].(map[string]any)
	if !ok {
		failLoad("$inverse needs an object with index or relation, and key")
	}
	for k := range spec {
		if k != "index" && k != "relation" && k != "key" {
			failLoad("$inverse: unknown key %q", k)
		}
	}
	_, hasIndex := spec["index"].(string)
	_, hasRel := spec["relation"].(string)
	if hasIndex == hasRel { // neither, or both
		failLoad("$inverse needs exactly one of index or relation, as a name")
	}
	key, has := spec["key"]
	if !has {
		failLoad("$inverse needs a key operand")
	}
	if isShaxonForm(key) {
		failLoad("$inverse.key: path-expression forms do not nest (F2)")
	}
	if e := jaxson.CheckOperandWith(StaticForms(), key, localNames(c)...); e != nil {
		failLoad("$inverse.key: %s", e.Msg)
	}
}

func checkClosureForm(name string, form map[string]any, c *jaxson.Checker) {
	for k := range form {
		if k != name && k != "from" && k != "maxDepth" {
			failLoad("%s: unknown key %q", name, k)
		}
	}
	locals := localNames(c)
	checkStep(name, form[name], locals)
	checkMaxDepth(name, form)
	if from, has := form["from"]; has {
		if isShaxonForm(from) {
			failLoad("%s.from: path-expression forms do not nest (F2)", name)
		}
		if e := jaxson.CheckPathWith(StaticForms(), from, locals...); e != nil {
			failLoad("%s.from: %s", name, e.Msg)
		}
		return
	}
	if !c.Locals["focus"] && !c.Locals["item"] {
		failLoad("%s: no from path, and no ambient position here (local.focus or local.item) to start from (F3)", name)
	}
}

// ---------------------------------------------------------------- evaluation

type formEnv struct {
	reg     *Registries
	idx     func() *IndexSet
	ambient *Ambient
}

// defs returns the forms with evaluation as well as checking.
func (env formEnv) defs() map[string]jaxson.FormDef {
	forms := StaticForms()
	set := func(name string, eval func(m *jaxson.Machine, f map[string]any) any) {
		d := forms[name]
		d.Eval = eval
		forms[name] = d
	}
	set("$altPath", env.evalAltPath)
	set("$inverse", env.evalInverse)
	for _, n := range closureForms {
		name := n
		set(name, func(m *jaxson.Machine, f map[string]any) any { return env.evalClosure(name, m, f) })
	}
	return forms
}

func (env formEnv) evalAltPath(m *jaxson.Machine, form map[string]any) any {
	for _, alt := range form["$altPath"].([]any) {
		root, segs := evalPath(m, "$altPath", alt.([]any))
		if v, err := m.Walk(root, segs); err == nil {
			return jaxson.Clone(v)
		}
	}
	jaxson.Fail("EXECUTION_ERROR", "MISSING_PATH", "no alternative of $altPath resolves")
	return nil
}

func (env formEnv) evalInverse(m *jaxson.Machine, form map[string]any) any {
	spec := form["$inverse"].(map[string]any)
	t := ReferenceTarget{}
	if name, ok := spec["index"].(string); ok {
		t.Index = name
	} else {
		t.Relation = spec["relation"].(string)
	}
	key := m.Eval(spec["key"])
	found := env.idx().Inverse(t, key)
	out := make([]any, len(found))
	for i, p := range found {
		out[i] = pathValue(p)
	}
	return out
}

func (env formEnv) evalClosure(name string, m *jaxson.Machine, form map[string]any) any {
	var base []any
	if from, has := form["from"]; has {
		root, segs := evalPath(m, name, from.([]any))
		base = fullPath(root, segs)
	} else if env.ambient != nil && env.ambient.Path() != nil {
		base = env.ambient.Path()
	} else {
		jaxson.Fail("PROGRAM_ERROR", "", "%s has no from path and no ambient position here (F3)", name)
	}
	depth := int(form["maxDepth"].(*big.Rat).Num().Int64())
	foci, overflow := closure(m, base, form[name].([]any), depth, name == "$path+")
	if overflow != nil {
		jaxson.Fail("EXECUTION_ERROR", CodePathDepthExceeded, "%s: %s (F4)", pathString(overflow), msgPathDepthExceeded)
	}
	out := make([]any, len(foci))
	for i, f := range foci {
		out[i] = pathValue(f.Path)
	}
	return out
}

// ---------------------------------------------------------------- name check

// checkInverseNames verifies, once everything is parsed (F6), that every
// `$inverse` in the package names a declared multi index or a declared
// relation.
func checkInverseNames(r *Registries) {
	check := func(ctx string, x any) {
		scanInverse(x, func(spec map[string]any) {
			if name, ok := spec["index"].(string); ok {
				ix, declared := r.Indices[name]
				if !declared {
					failLoad("%s: $inverse names an undeclared index %q", ctx, name)
				}
				if !ix.Multi {
					failLoad("%s: $inverse needs a multi index; %q is not", ctx, name)
				}
				return
			}
			name, _ := spec["relation"].(string)
			if _, declared := r.Relations[name]; !declared {
				failLoad("%s: $inverse names an undeclared relation %q", ctx, name)
			}
		})
	}
	for _, name := range jaxson.SortedKeys(anyKeys(r.Indices)) {
		ix := r.Indices[name]
		check("indices."+name+".source", ix.Source)
		check("indices."+name+".key", ix.Key)
	}
	for _, name := range jaxson.SortedKeys(anyKeys(r.Computes)) {
		c := r.Computes[name]
		check("computes."+name, c.With)
		check("computes."+name, c.Expr)
	}
	for _, name := range jaxson.SortedKeys(anyKeys(r.Shapes)) {
		eachShape(r.Shapes[name], func(s ShapeDecl) {
			for _, ck := range s.Check {
				check("shapes."+name+".check", ck.With)
				check("shapes."+name+".check", ck.Expr)
			}
			for _, of := range s.RefOf {
				check("shapes."+name+".of", of)
			}
		})
	}
}

func anyKeys[V any](m map[string]V) map[string]any {
	out := make(map[string]any, len(m))
	for k := range m {
		out[k] = nil
	}
	return out
}

// scanInverse calls fn with the argument of every `$inverse` form under x.
func scanInverse(x any, fn func(spec map[string]any)) {
	switch t := x.(type) {
	case map[string]any:
		if spec, ok := t["$inverse"].(map[string]any); ok {
			fn(spec)
		}
		for _, k := range jaxson.SortedKeys(t) {
			scanInverse(t[k], fn)
		}
	case []any:
		for _, e := range t {
			scanInverse(e, fn)
		}
	}
}

// eachShape visits s and every inline shape declared inside it.
func eachShape(s ShapeDecl, fn func(ShapeDecl)) {
	fn(s)
	visit := func(fd FieldDecl) {
		if fd.Inline != nil {
			eachShape(*fd.Inline, fn)
		}
	}
	for _, fd := range s.Fields {
		visit(fd)
	}
	if s.Items != nil {
		visit(*s.Items)
	}
	for _, q := range s.Qualified {
		visit(q.Target)
	}
	for _, group := range [][]FieldDecl{s.And, s.Or, s.Xone, s.Not} {
		for _, fd := range group {
			visit(fd)
		}
	}
}
