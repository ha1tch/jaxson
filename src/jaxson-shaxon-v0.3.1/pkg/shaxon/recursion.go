// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the Apache License, Version 2.0.
// https://www.apache.org/licenses/LICENSE-2.0
package shaxon

// Core section 2: limits.maxShapeDepth is mandatory whenever any shape in
// the package can recurse — through extends (already flattened into one
// shape by the time this runs, so it can't itself be a source of new
// graph edges here) or through a field/items/qualified/combinator member
// that names a shape reachable from itself. This is a directed-graph
// cycle check over named shapes; an inline shape is not a graph node —
// references inside it collapse onto its nearest enclosing named shape.

import "math/big"

func collectRefs(s ShapeDecl) []string {
	var refs []string
	add := func(fd FieldDecl) {
		if fd.ShapeRef != "" {
			refs = append(refs, fd.ShapeRef)
			return
		}
		if fd.Inline != nil {
			refs = append(refs, collectRefs(*fd.Inline)...)
		}
	}
	for _, fd := range s.Fields {
		add(fd)
	}
	if s.Items != nil {
		add(*s.Items)
	}
	for _, q := range s.Qualified {
		add(q.Target)
	}
	for _, fd := range s.And {
		add(fd)
	}
	for _, fd := range s.Or {
		add(fd)
	}
	for _, fd := range s.Xone {
		add(fd)
	}
	for _, fd := range s.Not {
		add(fd)
	}
	return refs
}

const (
	colorWhite = 0
	colorGray  = 1
	colorBlack = 2
)

func checkRecursionBound(pkg map[string]any, r *Registries) {
	color := map[string]int{}
	hasCycle := false
	var visit func(name string)
	visit = func(name string) {
		if hasCycle || color[name] == colorBlack {
			return
		}
		if color[name] == colorGray {
			hasCycle = true
			return
		}
		color[name] = colorGray
		if s, ok := r.Shapes[name]; ok {
			for _, ref := range collectRefs(s) {
				visit(ref)
				if hasCycle {
					return
				}
			}
		}
		color[name] = colorBlack
	}
	for name := range r.Shapes {
		visit(name)
		if hasCycle {
			break
		}
	}
	if !hasCycle {
		return
	}
	limits, _ := pkg["limits"].(map[string]any)
	v, has := limits["maxShapeDepth"]
	if !has {
		failLoad("a shape can recurse (via fields/items/qualified/combinators) but limits.maxShapeDepth is not declared")
	}
	n, ok := v.(*big.Rat)
	if !ok || !n.IsInt() || n.Sign() <= 0 {
		failLoad("limits.maxShapeDepth must be a positive integer")
	}
}
