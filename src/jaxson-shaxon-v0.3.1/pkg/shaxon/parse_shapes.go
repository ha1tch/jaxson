// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the Apache License, Version 2.0.
// https://www.apache.org/licenses/LICENSE-2.0
package shaxon

// Core section 4. KNOWN GAP, flagged rather than guessed: the merge
// table's "override": true / {"override": {...}} mechanism is described
// only in prose — no JSON example anywhere in core section 4 shows where
// an override key is actually written. This file implements the
// *default* aggregation column of the merge table in full (union fields,
// AND closed, concatenate and/or/xone, union required/requiredIds,
// accumulate not/check) but has no way yet to override a parent's
// member — deferred until its syntax is confirmed, not approximated.

import (
	"math/big"

	"github.com/ha1tch/jaxson/pkg/jaxson"
)

func parseShapes(pkg map[string]any, r *Registries) {
	raw, has := pkg["shapes"]
	if !has {
		return
	}
	m, ok := raw.(map[string]any)
	if !ok {
		failLoad("shapes must be an object")
	}
	rawShapes := map[string]map[string]any{}
	for _, name := range jaxson.SortedKeys(m) {
		sm, ok := m[name].(map[string]any)
		if !ok {
			failLoad("shapes.%s must be an object", name)
		}
		rawShapes[name] = sm
	}
	resolved := map[string]ShapeDecl{}
	inProgress := map[string]bool{}
	for _, name := range jaxson.SortedKeys(m) {
		r.Shapes[name] = resolveNamedShape(name, rawShapes, resolved, inProgress, r)
	}
	checkShorthandRedundancy(r)
}

// resolveNamedShape resolves shapes.<name>, including its extends chain,
// memoized in resolved and cycle-checked via inProgress. A genuine
// extends cycle (A extends B extends A) is unconditionally SHA_SHAPE_ERROR
// — settled earlier: extends resolves by static merging at load time, not
// per-focus-node recursion the way a self-referencing fields entry does,
// so no depth bound can make a literal cycle terminate correctly.
func resolveNamedShape(name string, rawShapes map[string]map[string]any, resolved map[string]ShapeDecl, inProgress map[string]bool, r *Registries) ShapeDecl {
	if s, done := resolved[name]; done {
		return s
	}
	if inProgress[name] {
		failLoad("shapes.%s: extends cycle", name)
	}
	raw, ok := rawShapes[name]
	if !ok {
		failLoad("shapes.%s: extends names an undeclared shape", name)
	}
	inProgress[name] = true
	own := parseShapeBody(name, raw, r)
	if parentName, has := raw["extends"]; has {
		pn, ok := parentName.(string)
		if !ok {
			failLoad("shapes.%s: extends must be a string", name)
		}
		parent := resolveNamedShape(pn, rawShapes, resolved, inProgress, r)
		own = mergeShapes(parent, own)
	}
	delete(inProgress, name)
	own.Name = name
	resolved[name] = own
	return own
}

func mergeShapes(parent, child ShapeDecl) ShapeDecl {
	out := child
	out.Closed = parent.Closed && child.Closed
	out.IgnoredProperties = append(append([]string{}, parent.IgnoredProperties...), child.IgnoredProperties...)

	fields := map[string]FieldDecl{}
	for k, v := range parent.Fields {
		fields[k] = v
	}
	for k, v := range child.Fields {
		if _, collide := parent.Fields[k]; collide {
			failLoad("shapes.%s: field %q collides with an extended shape's field", child.Name, k)
		}
		fields[k] = v
	}
	out.Fields = fields

	out.Required = unionStrings(parent.Required, child.Required)

	reqIds := map[string]string{}
	for k, v := range parent.RequiredIds {
		reqIds[k] = v
	}
	for k, v := range child.RequiredIds {
		reqIds[k] = v
	}
	out.RequiredIds = reqIds

	out.And = append(append([]FieldDecl{}, parent.And...), child.And...)
	out.Or = append(append([]FieldDecl{}, parent.Or...), child.Or...)
	out.Xone = append(append([]FieldDecl{}, parent.Xone...), child.Xone...)
	out.Not = append(append([]FieldDecl{}, parent.Not...), child.Not...)
	out.Check = append(append([]CombinatorCheck{}, parent.Check...), child.Check...)

	if child.Kind == "" {
		out.Kind = parent.Kind
	}
	if child.Items == nil {
		out.Items = parent.Items
	}
	switch {
	case child.Qualified == nil:
		out.Qualified = parent.Qualified
	case parent.Qualified == nil:
		out.Qualified = child.Qualified
	default:
		q := map[string]QualifiedDecl{}
		for k, v := range parent.Qualified {
			q[k] = v
		}
		for k, v := range child.Qualified {
			q[k] = v
		}
		out.Qualified = q
	}
	return out
}

func unionStrings(a, b []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range append(append([]string{}, a...), b...) {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

var shapeKeywordKeys = map[string]bool{
	"minLen": true, "maxLen": true, "enum": true, "min": true, "max": true,
	"int": true, "minItems": true, "maxItems": true,
}

var shapeStructuralKeys = map[string]bool{
	"kind": true, "closed": true, "ignoredProperties": true, "fields": true,
	"required": true, "requiredIds": true, "and": true, "or": true, "xone": true,
	"not": true, "check": true, "severity": true, "message": true, "extends": true,
	"items": true, "qualified": true,
	"index": true, "relation": true, "of": true, "by": true,
}

// parseShapeBody parses everything a shape declares itself, not
// resolving extends (the caller does that). name is used only for error
// messages; the returned ShapeDecl.Name is set by the caller once fully
// resolved.
func parseShapeBody(name string, raw map[string]any, r *Registries) ShapeDecl {
	for k := range raw {
		if !shapeStructuralKeys[k] && !shapeKeywordKeys[k] {
			failLoad("shapes.%s: unknown key %q", name, k)
		}
	}

	var s ShapeDecl
	if kv, has := raw["kind"]; has {
		k, ok := kv.(string)
		if !ok || !validKinds[k] {
			failLoad("shapes.%s: unknown kind %v", name, kv)
		}
		s.Kind = k
	}

	// Core section 4: "closed objects are the default posture" — absence
	// of the key means true, not Go's zero value. Getting this backwards
	// would make the extends AND-merge silently collapse to false for any
	// child that simply doesn't repeat "closed": true (caught by
	// TestParseRegistries_ExtendsMerge).
	s.Closed = true
	if cv, has := raw["closed"]; has {
		b, ok := cv.(bool)
		if !ok {
			failLoad("shapes.%s: closed must be a boolean", name)
		}
		s.Closed = b
	}
	if ipv, has := raw["ignoredProperties"]; has {
		arr, ok := ipv.([]any)
		if !ok {
			failLoad("shapes.%s: ignoredProperties must be an array", name)
		}
		for _, e := range arr {
			es, ok := e.(string)
			if !ok {
				failLoad("shapes.%s: ignoredProperties entries must be strings", name)
			}
			s.IgnoredProperties = append(s.IgnoredProperties, es)
		}
	}

	if fv, has := raw["fields"]; has {
		fm, ok := fv.(map[string]any)
		if !ok {
			failLoad("shapes.%s: fields must be an object", name)
		}
		s.Fields = map[string]FieldDecl{}
		for _, fname := range jaxson.SortedKeys(fm) {
			s.Fields[fname] = parseField(name+"."+fname, fm[fname], r)
		}
	}

	if iv, has := raw["items"]; has {
		fd := parseField(name+".items", iv, r)
		s.Items = &fd
	}

	if rv, has := raw["required"]; has {
		arr, ok := rv.([]any)
		if !ok {
			failLoad("shapes.%s: required must be an array", name)
		}
		for _, e := range arr {
			es, ok := e.(string)
			if !ok {
				failLoad("shapes.%s: required entries must be strings", name)
			}
			s.Required = append(s.Required, es)
		}
	}
	if riv, has := raw["requiredIds"]; has {
		rim, ok := riv.(map[string]any)
		if !ok {
			failLoad("shapes.%s: requiredIds must be an object", name)
		}
		reqSet := map[string]bool{}
		for _, rq := range s.Required {
			reqSet[rq] = true
		}
		s.RequiredIds = map[string]string{}
		for _, k := range jaxson.SortedKeys(rim) {
			id, ok := rim[k].(string)
			if !ok {
				failLoad("shapes.%s: requiredIds.%s must be a string", name, k)
			}
			if !reqSet[k] {
				failLoad("shapes.%s: requiredIds names %q, which is not in required", name, k)
			}
			s.RequiredIds[k] = id
		}
	}

	for _, combKey := range []string{"and", "or", "xone"} {
		cv, has := raw[combKey]
		if !has {
			continue
		}
		arr, ok := cv.([]any)
		if !ok {
			failLoad("shapes.%s: %s must be an array", name, combKey)
		}
		var list []FieldDecl
		for _, e := range arr {
			list = append(list, parseField(name+"."+combKey+"[]", e, r))
		}
		switch combKey {
		case "and":
			s.And = list
		case "or":
			s.Or = list
		case "xone":
			s.Xone = list
		}
	}
	if nv, has := raw["not"]; has {
		s.Not = []FieldDecl{parseField(name+".not", nv, r)}
	}

	if qv, has := raw["qualified"]; has {
		s.Qualified = map[string]QualifiedDecl{"": parseQualified(name, qv, r)}
	}

	if cv, has := raw["check"]; has {
		s.Check = []CombinatorCheck{parseCheck(name, cv, r)}
	}

	if sv, has := raw["severity"]; has {
		sev, ok := sv.(string)
		if !ok || !validSeverities[sev] {
			failLoad("shapes.%s: severity must be violation, warning, or info", name)
		}
		s.Severity = sev
	} else {
		s.Severity = "violation"
	}
	if mv, has := raw["message"]; has {
		msg, ok := mv.(string)
		if !ok {
			failLoad("shapes.%s: message must be a string", name)
		}
		s.Message = msg
	}

	if s.Kind == "reference" {
		parseReferenceMembers(name, raw, &s, r)
	} else {
		for _, k := range []string{"index", "relation", "of", "by"} {
			if _, has := raw[k]; has {
				failLoad("shapes.%s: %q is only valid on a reference-kind shape", name, k)
			}
		}
	}

	parseKeywords(name, raw, &s)
	return s
}

func parseQualified(ctx string, v any, r *Registries) QualifiedDecl {
	qm, ok := v.(map[string]any)
	if !ok {
		failLoad("%s.qualified must be an object", ctx)
	}
	for k := range qm {
		if k != "shape" && k != "min" && k != "max" {
			failLoad("%s.qualified: unknown key %q", ctx, k)
		}
	}
	sv, has := qm["shape"]
	if !has {
		failLoad("%s.qualified: needs shape", ctx)
	}
	var target FieldDecl
	switch t := sv.(type) {
	case string:
		target = FieldDecl{ShapeRef: t}
	case map[string]any:
		inline := parseShapeBody(ctx+".qualified.shape", t, r)
		target = FieldDecl{Inline: &inline}
	default:
		failLoad("%s.qualified.shape must be a shape name or an inline shape", ctx)
	}
	q := QualifiedDecl{Target: target}
	if mv, has := qm["min"]; has {
		n := mustNonNegInt(ctx+".qualified.min", mv)
		q.Min = &n
	}
	if mv, has := qm["max"]; has {
		n := mustNonNegInt(ctx+".qualified.max", mv)
		q.Max = &n
	}
	return q
}

func parseCheck(ctx string, v any, r *Registries) CombinatorCheck {
	cm, ok := v.(map[string]any)
	if !ok {
		failLoad("%s.check must be an object", ctx)
	}
	_, hasCompute := cm["compute"]
	id := ""
	if idv, has := cm["id"]; has {
		ids, ok := idv.(string)
		if !ok {
			failLoad("%s.check.id must be a string", ctx)
		}
		id = ids
	}
	if hasCompute {
		for k := range cm {
			if k != "compute" && k != "id" {
				failLoad("%s.check: the compute form cannot also declare %q", ctx, k)
			}
		}
		cn, ok := cm["compute"].(string)
		if !ok {
			failLoad("%s.check.compute must be a string", ctx)
		}
		if _, declared := r.Computes[cn]; !declared {
			failLoad("%s.check: compute names an undeclared computes entry %q", ctx, cn)
		}
		return CombinatorCheck{ComputeRef: cn, ID: id}
	}
	for k := range cm {
		if k != "with" && k != "expr" && k != "id" {
			failLoad("%s.check: unknown key %q", ctx, k)
		}
	}
	expr, hasExpr := cm["expr"]
	if !hasExpr {
		failLoad("%s.check: needs an expr", ctx)
	}
	with, _ := cm["with"].(map[string]any)
	if _, present := cm["with"]; present && with == nil {
		failLoad("%s.check.with must be an object", ctx)
	}
	checkComputeBody(with, expr, ctx+".check")
	return CombinatorCheck{With: with, Expr: expr, ID: id}
}

func parseReferenceMembers(name string, raw map[string]any, s *ShapeDecl, r *Registries) {
	_, hasIndex := raw["index"]
	_, hasRelation := raw["relation"]
	_, hasOf := raw["of"]
	_, hasBy := raw["by"]
	formCount := 0
	if hasIndex {
		formCount++
	}
	if hasRelation {
		formCount++
	}
	if hasOf || hasBy {
		formCount++
	}
	if formCount != 1 {
		failLoad("shapes.%s: a reference-kind shape needs exactly one of index, relation, or of+by", name)
	}
	switch {
	case hasIndex:
		iv, ok := raw["index"].(string)
		if !ok {
			failLoad("shapes.%s.index must be a string", name)
		}
		if _, declared := r.Indices[iv]; !declared {
			failLoad("shapes.%s: index names an undeclared index %q", name, iv)
		}
		s.RefIndex = iv
	case hasRelation:
		rv, ok := raw["relation"].(string)
		if !ok {
			failLoad("shapes.%s.relation must be a string", name)
		}
		if _, declared := r.Relations[rv]; !declared {
			failLoad("shapes.%s: relation names an undeclared relation %q", name, rv)
		}
		s.RefRelation = rv
	default:
		if !hasOf || !hasBy {
			failLoad("shapes.%s: reference of/by must be declared together", name)
		}
		of := raw["of"]
		if e := jaxson.CheckOperand(of); e != nil {
			failLoad("shapes.%s.of: %s", name, e.Msg)
		}
		byArr, ok := raw["by"].([]any)
		if !ok || len(byArr) == 0 {
			failLoad("shapes.%s.by must be a non-empty array", name)
		}
		for _, seg := range byArr {
			if _, ok := seg.(string); !ok {
				failLoad("shapes.%s.by: every segment must be a plain field name", name)
			}
		}
		s.RefOf, s.RefBy = []any{of}, byArr
	}
}

func parseKeywords(name string, raw map[string]any, s *ShapeDecl) {
	if ev, has := raw["enum"]; has {
		arr, ok := ev.([]any)
		if !ok {
			failLoad("shapes.%s: enum must be an array", name)
		}
		if s.Kind != "" && !enumKinds[s.Kind] {
			failLoad("shapes.%s: enum is not legal on kind %q (only string/number/boolean/null)", name, s.Kind)
		}
		s.Keywords.Enum = arr
	}
	if v, has := raw["minLen"]; has {
		n := mustNonNegInt(name+".minLen", v)
		s.Keywords.MinLen = &n
	}
	if v, has := raw["maxLen"]; has {
		n := mustNonNegInt(name+".maxLen", v)
		s.Keywords.MaxLen = &n
	}
	if v, has := raw["minItems"]; has {
		n := mustNonNegInt(name+".minItems", v)
		s.Keywords.MinItems = &n
	}
	if v, has := raw["maxItems"]; has {
		n := mustNonNegInt(name+".maxItems", v)
		s.Keywords.MaxItems = &n
	}
	if v, has := raw["min"]; has {
		f := mustNumber(name+".min", v)
		s.Keywords.Min = &f
	}
	if v, has := raw["max"]; has {
		f := mustNumber(name+".max", v)
		s.Keywords.Max = &f
	}
	if v, has := raw["int"]; has {
		b, ok := v.(bool)
		if !ok {
			failLoad("shapes.%s.int must be a boolean", name)
		}
		s.Keywords.WantInt = b
	}
}

// parseField parses one field/items/combinator-member value: either
// {"shape": "Name"} or an inline shape declaration.
func parseField(ctx string, v any, r *Registries) FieldDecl {
	m, ok := v.(map[string]any)
	if !ok {
		failLoad("%s must be an object", ctx)
	}
	if sv, has := m["shape"]; has && len(m) == 1 {
		if sn, ok := sv.(string); ok {
			return FieldDecl{ShapeRef: sn}
		}
	}
	inline := parseShapeBody(ctx, m, r)
	return FieldDecl{Inline: &inline}
}

func mustNonNegInt(ctx string, v any) int64 {
	r, ok := v.(*big.Rat)
	if !ok || !r.IsInt() || r.Sign() < 0 {
		failLoad("%s must be a non-negative integer", ctx)
	}
	return r.Num().Int64()
}

func mustNumber(ctx string, v any) float64 {
	r, ok := v.(*big.Rat)
	if !ok {
		failLoad("%s must be a number", ctx)
	}
	f, _ := r.Float64()
	return f
}

// checkShorthandRedundancy is the second half of section 4c: an inline
// reference of/by shorthand whose normalised (of, by) matches a
// relation's to side.
func checkShorthandRedundancy(r *Registries) {
	for _, shapeName := range jaxson.SortedKeys(shapeNameSet(r)) {
		walkFields(r.Shapes[shapeName], func(fieldCtx string, fd FieldDecl) {
			if fd.Inline == nil || len(fd.Inline.RefOf) == 0 {
				return
			}
			of := fd.Inline.RefOf[0]
			by := fd.Inline.RefBy
			opAsPath, ok := of.(map[string]any)
			if !ok || len(by) != 1 {
				return
			}
			p, ok := opAsPath["$path"].([]any)
			if !ok {
				return
			}
			for relName, rel := range r.Relations {
				if pathsEqual(p, rel.ToPath) && by[0] == rel.ToKey {
					failLoad("shapes.%s: inline reference shorthand duplicates relations.%s's to side — write {\"kind\": \"reference\", \"relation\": %q} instead", fieldCtx, relName, relName)
				}
			}
		})
	}
}

func shapeNameSet(r *Registries) map[string]any {
	out := map[string]any{}
	for k := range r.Shapes {
		out[k] = nil
	}
	return out
}

// walkFields visits every inline field/combinator-member of s.
func walkFields(s ShapeDecl, fn func(ctx string, fd FieldDecl)) {
	for name, fd := range s.Fields {
		fn(name, fd)
		if fd.Inline != nil {
			walkFields(*fd.Inline, fn)
		}
	}
	for _, fd := range append(append(append([]FieldDecl{}, s.And...), s.Or...), s.Xone...) {
		if fd.Inline != nil {
			walkFields(*fd.Inline, fn)
		}
	}
	for _, fd := range s.Not {
		if fd.Inline != nil {
			walkFields(*fd.Inline, fn)
		}
	}
	if s.Items != nil && s.Items.Inline != nil {
		walkFields(*s.Items.Inline, fn)
	}
}
