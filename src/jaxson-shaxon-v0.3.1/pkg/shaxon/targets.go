// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the Apache License, Version 2.0.
// https://www.apache.org/licenses/LICENSE-2.0
package shaxon

// Phase 4.2 (plan section 7): targets. A target resolves to an ordered list
// of focus nodes, each a path and the value found there. Core section 7.
//
// ====================================================================
// Decisions where the spec is silent or ambiguous (T1..T5)
// ====================================================================
//
//	T1  Cost. A target is one `target.resolve` event, charged before the
//	    focus nodes are visited, with the size of what it will scan: 1 for
//	    `$path`, the collection's length for `$each` and `$discriminator`,
//	    the number of source elements for `$indexed`. Under the `unit` table
//	    that is one step per target. The index build `$indexed` needs is
//	    charged separately, as `index.element`, by the IndexSet.
//	T2  `$each` over something that is neither an array nor an object is a
//	    TYPE_ERROR. (Evaluator.each, which walks nested collections inside a
//	    shape, skips non-collections; a target names its population on
//	    purpose, so a wrong kind is a mistake worth raising.)
//	T3  `$discriminator` examines every element of `at`; an element that is
//	    not an object, or has no `field`, is not in the population. The
//	    comparison is jaxson.Equal, so 1 and 1.0 match.
//	T4  Objects are walked in code-point key order, arrays in index order
//	    (core section 7, last bullet but one).
//	T5  A target path may contain operand segments (so a `check` inside a
//	    `for` can name local.i); they are evaluated when the target is
//	    resolved, and must give a string or a non-negative integer.

import (
	"math/big"

	"github.com/ha1tch/jaxson/pkg/jaxson"
)

// anyLocal, passed as the only entry of parseTarget's locals, skips the
// static check of operand segments: the `check` instruction has checked them
// against the locals in scope already, and at run time they are bound.
const anyLocal = "*"

// EventTargetResolve is the cost-table event for resolving one target.
const EventTargetResolve = "target.resolve"

// Target kinds.
const (
	TargetPath          = "path"
	TargetEach          = "each"
	TargetDiscriminator = "discriminator"
	TargetIndexed       = "indexed"
	TargetClosure       = "closure"
)

// Target is a parsed target form (core section 7).
type Target struct {
	Kind  string
	Path  []any  // path, each: the path named; discriminator: `at`
	Field string // discriminator: the field compared
	Value any    // discriminator: the value it must equal
	Index string // indexed: the index name

	// closure ($path* / $path+): Path is the `from` base.
	Step     []any
	MaxDepth int
	Plus     bool // $path+ rather than $path*
}

// Focus is one focus node: where it is and what is there.
type Focus struct {
	Path  []any // root first, then segments (ints for array positions)
	Value any
}

// parseTarget statically checks a target form. locals lists the `local.*`
// names in scope for operand segments (none in a package-level `validate`).
func parseTarget(ctx string, raw any, r *Registries, locals ...string) Target {
	mp, ok := raw.(map[string]any)
	if !ok {
		failLoad("%s: a target must be an object", ctx)
	}
	for _, form := range []string{"$path*", "$path+"} {
		if _, is := mp[form]; is {
			return parseClosure(ctx, form, mp, locals)
		}
	}
	if len(mp) != 1 {
		failLoad("%s: a target must have exactly one form", ctx)
	}
	for form, v := range mp {
		switch form {
		case "$path":
			return Target{Kind: TargetPath, Path: targetPath(ctx+".$path", v, locals)}
		case "$each":
			return Target{Kind: TargetEach, Path: targetPath(ctx+".$each", v, locals)}
		case "$discriminator":
			d, ok := v.(map[string]any)
			if !ok {
				failLoad("%s.$discriminator must be an object", ctx)
			}
			for k := range d {
				if k != "at" && k != "field" && k != "value" {
					failLoad("%s.$discriminator: unknown key %q", ctx, k)
				}
			}
			field, ok := d["field"].(string)
			if !ok || field == "" {
				failLoad("%s.$discriminator: field must be a plain field name", ctx)
			}
			val, has := d["value"]
			if !has {
				failLoad("%s.$discriminator: needs a value", ctx)
			}
			if e := jaxson.CheckOperand(map[string]any{"$lit": val}); e != nil {
				failLoad("%s.$discriminator.value: %s", ctx, e.Msg)
			}
			at, has := d["at"]
			if !has {
				failLoad("%s.$discriminator: needs an at path", ctx)
			}
			return Target{Kind: TargetDiscriminator, Path: targetPath(ctx+".$discriminator.at", at, locals), Field: field, Value: val}
		case "$indexed":
			name, ok := v.(string)
			if !ok {
				failLoad("%s.$indexed must be an index name", ctx)
			}
			if _, declared := r.Indices[name]; !declared {
				failLoad("%s.$indexed names an undeclared index %q", ctx, name)
			}
			return Target{Kind: TargetIndexed, Index: name}
		}
		failLoad("%s: unknown target form %q", ctx, form)
	}
	return Target{}
}

func targetPath(ctx string, v any, locals []string) []any {
	p, ok := v.([]any)
	if !ok || len(p) == 0 {
		failLoad("%s must be a path array", ctx)
	}
	if root, _ := p[0].(string); root != "input" && root != "state" && root != "output" {
		failLoad("%s: a target path starts at input, state or output", ctx)
	}
	if len(locals) == 1 && locals[0] == anyLocal {
		return p // already checked statically; at run time the locals are bound
	}
	if e := jaxson.CheckPath(p, locals...); e != nil {
		failLoad("%s: %s", ctx, e.Msg)
	}
	return p
}

// evalPath evaluates a path's operand segments (T5), returning the root and
// the segments as strings and ints.
func evalPath(m *jaxson.Machine, label string, p []any) (string, []any) {
	root := p[0].(string)
	segs := make([]any, 0, len(p)-1)
	for _, seg := range p[1:] {
		v := seg
		if mp, ok := seg.(map[string]any); ok {
			v = m.Eval(mp)
		}
		switch u := v.(type) {
		case string:
			segs = append(segs, u)
		case *big.Rat:
			if !u.IsInt() || u.Sign() < 0 || !u.Num().IsInt64() {
				jaxson.Fail("EXECUTION_ERROR", "TYPE_ERROR", "%s: a path segment must be a string or a non-negative integer, got %s", label, jaxson.Show(u))
			}
			segs = append(segs, int(u.Num().Int64()))
		default:
			jaxson.Fail("EXECUTION_ERROR", "TYPE_ERROR", "%s: a path segment must be a string or a non-negative integer, got %s", label, jaxson.TypeName(v))
		}
	}
	return root, segs
}

// walkOrRaise reads a value without cloning it, raising the Machine's own
// MISSING_PATH / TYPE_ERROR when the path does not resolve.
func walkOrRaise(m *jaxson.Machine, root string, segs []any) any {
	v, err := m.Walk(root, segs)
	if err != nil {
		panic(err)
	}
	return v
}

func fullPath(root string, segs []any) []any {
	return append([]any{root}, segs...)
}

// resolve returns the target's focus nodes in visiting order, charging the
// `target.resolve` event first (T1). idx supplies `$indexed`. depthAt is the
// path of the node at which a closure hit its maxDepth horizon with the
// chain still going (C3), or nil.
func (t Target) resolve(m *jaxson.Machine, idx *IndexSet) (foci []Focus, depthAt []any) {
	switch t.Kind {
	case TargetClosure:
		root, segs := evalPath(m, "target", t.Path)
		m.ChargeEvent(EventTargetResolve, 1)
		return closure(m, fullPath(root, segs), t.Step, t.MaxDepth, t.Plus)

	case TargetPath:
		root, segs := evalPath(m, "target", t.Path)
		v := walkOrRaise(m, root, segs)
		m.ChargeEvent(EventTargetResolve, 1)
		return []Focus{{Path: fullPath(root, segs), Value: v}}, nil

	case TargetEach, TargetDiscriminator:
		root, segs := evalPath(m, "target", t.Path)
		coll := walkOrRaise(m, root, segs)
		base := fullPath(root, segs)
		var all []Focus
		switch c := coll.(type) {
		case []any:
			m.ChargeEvent(EventTargetResolve, int64(len(c)))
			for i, el := range c {
				all = append(all, Focus{Path: appendPath(base, i), Value: el})
			}
		case *jaxson.Object:
			m.ChargeEvent(EventTargetResolve, int64(c.Len()))
			for _, k := range c.SortedKeys() {
				v, _ := c.Get(k)
				all = append(all, Focus{Path: appendPath(base, k), Value: v})
			}
		default:
			jaxson.Fail("EXECUTION_ERROR", "TYPE_ERROR", "%s: a target collection must be an array or an object, got %s", pathString(base), jaxson.TypeName(coll))
		}
		if t.Kind == TargetEach {
			return all, nil
		}
		var out []Focus
		for _, f := range all {
			if obj, ok := f.Value.(*jaxson.Object); ok {
				if got, has := obj.Get(t.Field); has && jaxson.Equal(got, t.Value) {
					out = append(out, f)
				}
			}
		}
		return out, nil

	case TargetIndexed:
		paths := idx.Elements(t.Index)
		m.ChargeEvent(EventTargetResolve, int64(len(paths)))
		out := make([]Focus, 0, len(paths))
		for _, p := range paths {
			out = append(out, Focus{Path: p, Value: walkOrRaise(m, p[0].(string), p[1:])})
		}
		return out, nil
	}
	return nil, nil
}
