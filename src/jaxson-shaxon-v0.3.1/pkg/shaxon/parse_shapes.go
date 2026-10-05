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
// extends cycle (A extends B extends A) is unconditionally SHAX_SHAPE_ERROR
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
	validateKeywordsAgainstKind(name, own)
	validateRequiredIds(name, own)
	resolved[name] = own
	return own
}

func mergeShapes(parent, child ShapeDecl) ShapeDecl {
	out := child

	// Core section 4: closed accumulates as "most restrictive of parent/child
	// wins": a closed parent or a closed child makes the merged shape closed,
	// unless the child marks closedOverride. A child that does not write
	// "closed" at all inherits the parent's posture instead of imposing the
	// default (S8; a boolean AND of the two flags picked the most permissive).
	switch {
	case child.ClosedOverride:
		out.Closed = child.Closed
	case child.ClosedSet:
		out.Closed = parent.Closed || child.Closed
	default:
		out.Closed = parent.Closed
	}
	out.IgnoredProperties = append(append([]string{}, parent.IgnoredProperties...), child.IgnoredProperties...)

	fields := map[string]FieldDecl{}
	for k, v := range parent.Fields {
		fields[k] = v
	}
	for k, v := range child.Fields {
		if _, collide := parent.Fields[k]; collide && !v.Override {
			failLoad("shapes.%s: field %q collides with an extended shape's field (add \"override\": true to replace it)", child.Name, k)
		}
		fields[k] = v
	}
	out.Fields = fields

	out.Required = unionStrings(parent.Required, child.Required)

	reqIds := map[string]RequiredIdEntry{}
	for k, v := range parent.RequiredIds {
		reqIds[k] = v
	}
	for k, v := range child.RequiredIds {
		if _, collide := parent.RequiredIds[k]; collide && !v.Override {
			failLoad("shapes.%s: requiredIds %q collides with an extended shape's requiredIds (add \"override\": true to replace it)", child.Name, k)
		}
		reqIds[k] = v
	}
	out.RequiredIds = reqIds

	if child.AndOverride {
		out.And = child.And
	} else {
		out.And = append(append([]FieldDecl{}, parent.And...), child.And...)
	}
	if child.OrOverride {
		out.Or = child.Or
	} else {
		out.Or = append(append([]FieldDecl{}, parent.Or...), child.Or...)
	}
	if child.XoneOverride {
		out.Xone = child.Xone
	} else {
		out.Xone = append(append([]FieldDecl{}, parent.Xone...), child.Xone...)
	}
	if child.NotOverride {
		out.Not = child.Not
	} else {
		out.Not = append(append([]FieldDecl{}, parent.Not...), child.Not...)
	}
	if child.CheckOverride != nil {
		out.Check = []CombinatorCheck{*child.CheckOverride}
	} else {
		out.Check = append(append([]CombinatorCheck{}, parent.Check...), child.Check...)
	}

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
	"items": true, "qualified": true, "id": true,
	"index": true, "relation": true, "of": true, "by": true,
	// Whole-member extends-override flags (core section 4). "override"
	// itself is NOT here: it's valid only inside a fields entry, handled
	// by parseField directly, not as a general shape-body key.
	"closedOverride": true, "andOverride": true, "orOverride": true,
	"xoneOverride": true, "notOverride": true, "checkOverride": true,
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
		s.ClosedSet = true
	}
	if ov, has := raw["closedOverride"]; has {
		b, ok := ov.(bool)
		if !ok {
			failLoad("shapes.%s.closedOverride must be a boolean", name)
		}
		s.ClosedOverride = b
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
			s.Fields[fname] = parseField(name+"."+fname, fm[fname], r, true)
		}
	}

	if iv, has := raw["items"]; has {
		fd := parseField(name+".items", iv, r, false)
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
		// NOT validated against s.Required here — s.Required is this
		// shape's own pre-merge list, but requiredIds is legal whenever
		// the name is in the MERGED required list (e.g. a child adding
		// requiredIds for a name only its parent requires). See
		// validateRequiredIds, called post-merge, same reasoning as
		// validateKeywordsAgainstKind.
		s.RequiredIds = map[string]RequiredIdEntry{}
		for _, k := range jaxson.SortedKeys(rim) {
			s.RequiredIds[k] = parseRequiredIdEntry(name, k, rim[k])
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
			list = append(list, parseField(name+"."+combKey+"[]", e, r, false))
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
	for _, ov := range []struct {
		key string
		set func(bool)
	}{
		{"andOverride", func(b bool) { s.AndOverride = b }},
		{"orOverride", func(b bool) { s.OrOverride = b }},
		{"xoneOverride", func(b bool) { s.XoneOverride = b }},
		{"notOverride", func(b bool) { s.NotOverride = b }},
	} {
		if v, has := raw[ov.key]; has {
			b, ok := v.(bool)
			if !ok {
				failLoad("shapes.%s.%s must be a boolean", name, ov.key)
			}
			ov.set(b)
		}
	}
	if nv, has := raw["not"]; has {
		s.Not = []FieldDecl{parseField(name+".not", nv, r, false)}
	}

	if qv, has := raw["qualified"]; has {
		s.Qualified = map[string]QualifiedDecl{"": parseQualified(name, qv, r)}
	}

	if cv, has := raw["check"]; has {
		s.Check = []CombinatorCheck{parseCheck(name, cv, r)}
	}
	if ov, has := raw["checkOverride"]; has {
		cc := parseCheck(name+".checkOverride", ov, r)
		s.CheckOverride = &cc
		if _, hasCheck := raw["check"]; hasCheck {
			failLoad("shapes.%s: check and checkOverride cannot both be declared on the same shape — checkOverride replaces what the shape would otherwise AND with the parent, it is not an additional check", name)
		}
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

	if iv, has := raw["id"]; has {
		id, ok := iv.(string)
		if !ok {
			failLoad("shapes.%s: id must be a string", name)
		}
		s.ID = id
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
		if _, has := t["extends"]; has {
			failLoad("%s.qualified.shape: extends is only valid on a named, top-level shape", ctx)
		}
		inline := parseShapeBody(ctx+".qualified.shape", t, r)
		validateKeywordsAgainstKind(ctx+".qualified.shape", inline)
		validateRequiredIds(ctx+".qualified.shape", inline)
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
		if e := jaxson.CheckOperandWith(StaticForms(), of); e != nil {
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

// keywordLegalOn mirrors jaxson's own schemaKeys table (schema.go) — the
// same per-kind keyword vocabulary jaxson's inputSchema enforces, applied
// here to a shape declaration instead of a schema one. enum is handled
// separately (enumKinds), since it spans four kinds, not one.
var keywordLegalOn = map[string]string{
	"minLen": "string", "maxLen": "string",
	"min": "number", "max": "number", "int": "number",
	"minItems": "array", "maxItems": "array",
}

// parseKeywords extracts keyword values and checks each is well-formed in
// isolation (a non-negative integer, a number, a boolean). It does NOT
// check legality against Kind or cross-keyword ordering (min<=max etc.):
// both of those must wait until after extends has resolved Kind
// inheritance — see validateKeywordsAgainstKind, called post-merge.
func parseKeywords(name string, raw map[string]any, s *ShapeDecl) {
	if ev, has := raw["enum"]; has {
		arr, ok := ev.([]any)
		if !ok {
			failLoad("shapes.%s: enum must be an array", name)
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
		s.Keywords.MinRat = v.(*big.Rat)
	}
	if v, has := raw["max"]; has {
		f := mustNumber(name+".max", v)
		s.Keywords.Max = &f
		s.Keywords.MaxRat = v.(*big.Rat)
	}
	if v, has := raw["int"]; has {
		b, ok := v.(bool)
		if !ok {
			failLoad("shapes.%s.int must be a boolean", name)
		}
		s.Keywords.WantInt = b
	}
}

// validateRequiredIds checks every requiredIds key names a member in the
// final, post-merge Required list. Called wherever validateKeywordsAgainstKind
// is — same ordering reasoning: Required can itself come from a parent.
func validateRequiredIds(name string, s ShapeDecl) {
	reqSet := map[string]bool{}
	for _, rq := range s.Required {
		reqSet[rq] = true
	}
	for _, k := range jaxson.SortedKeys(toAnyMap(s.RequiredIds)) {
		if !reqSet[k] {
			failLoad("shapes.%s: requiredIds names %q, which is not in required", name, k)
		}
	}
}

func toAnyMap(m map[string]RequiredIdEntry) map[string]any {
	out := make(map[string]any, len(m))
	for k := range m {
		out[k] = nil
	}
	return out
}

// validateKeywordsAgainstKind checks keyword-vs-kind legality (mirroring
// jaxson's own schemaKeys table) and cross-keyword ordering. Called once
// per shape with its FINAL kind already known: immediately after parsing
// for an inline shape (which cannot extend — see parseField), and after
// extends has resolved for a named one (see resolveNamedShape).
func validateKeywordsAgainstKind(name string, s ShapeDecl) {
	for kw, wantKind := range keywordLegalOn {
		present := false
		switch kw {
		case "minLen":
			present = s.Keywords.MinLen != nil
		case "maxLen":
			present = s.Keywords.MaxLen != nil
		case "min":
			present = s.Keywords.Min != nil
		case "max":
			present = s.Keywords.Max != nil
		case "int":
			present = s.Keywords.WantInt
		case "minItems":
			present = s.Keywords.MinItems != nil
		case "maxItems":
			present = s.Keywords.MaxItems != nil
		}
		if present && s.Kind != wantKind {
			failLoad("shapes.%s: %q is not legal on kind %q (only %q)", name, kw, s.Kind, wantKind)
		}
	}
	if len(s.Keywords.Enum) > 0 && s.Kind != "" && !enumKinds[s.Kind] {
		failLoad("shapes.%s: enum is not legal on kind %q (only string/number/boolean/null)", name, s.Kind)
	}
	if s.Keywords.MinLen != nil && s.Keywords.MaxLen != nil && *s.Keywords.MinLen > *s.Keywords.MaxLen {
		failLoad("shapes.%s: minLen (%d) exceeds maxLen (%d)", name, *s.Keywords.MinLen, *s.Keywords.MaxLen)
	}
	if s.Keywords.MinItems != nil && s.Keywords.MaxItems != nil && *s.Keywords.MinItems > *s.Keywords.MaxItems {
		failLoad("shapes.%s: minItems (%d) exceeds maxItems (%d)", name, *s.Keywords.MinItems, *s.Keywords.MaxItems)
	}
	if s.Keywords.Min != nil && s.Keywords.Max != nil && *s.Keywords.Min > *s.Keywords.Max {
		failLoad("shapes.%s: min (%v) exceeds max (%v)", name, *s.Keywords.Min, *s.Keywords.Max)
	}
}

// parseField parses one field/items/combinator-member value: either
// {"shape": "Name"} (optionally with "override") or an inline shape
// declaration (optionally with "override" mixed into its own keys).
// allowOverride is true only when called from the fields map loop —
// per-item "override" has meaning only there (core section 4: "per-field
// override:true"); everywhere else (items/qualified/and/or/xone/not) only
// the whole-member fooOverride sibling keys apply, so "override" is
// rejected as an unknown key rather than silently accepted and ignored.
//
// extends is not legal on an inline shape — every spec example uses it
// only on a named, top-level shape, and disallowing it here removes an
// otherwise-real ordering hazard: a field's Kind can be known immediately
// after parsing only if it never depends on resolving another shape's
// extends chain.
func parseField(ctx string, v any, r *Registries, allowOverride bool) FieldDecl {
	m, ok := v.(map[string]any)
	if !ok {
		failLoad("%s must be an object", ctx)
	}
	override := false
	if ov, has := m["override"]; has {
		if !allowOverride {
			failLoad("%s: \"override\" is only valid inside a fields entry", ctx)
		}
		b, ok := ov.(bool)
		if !ok {
			failLoad("%s.override must be a boolean", ctx)
		}
		override = b
	}
	if sv, hasShape := m["shape"]; hasShape {
		rest := 0
		for k := range m {
			if k != "shape" && k != "override" {
				rest++
			}
		}
		if rest == 0 {
			if sn, ok := sv.(string); ok {
				return FieldDecl{ShapeRef: sn, Override: override}
			}
		}
	}
	if _, has := m["extends"]; has {
		failLoad("%s: extends is only valid on a named, top-level shape", ctx)
	}
	body := m
	if _, has := m["override"]; has {
		body = map[string]any{}
		for k, vv := range m {
			if k != "override" {
				body[k] = vv
			}
		}
	}
	inline := parseShapeBody(ctx, body, r)
	validateKeywordsAgainstKind(ctx, inline)
	validateRequiredIds(ctx, inline)
	return FieldDecl{Inline: &inline, Override: override}
}

// parseRequiredIdEntry accepts either a bare string ("id": "SOME_ID") or
// the promoted object form ("id": {"value": "SOME_ID", "override": true}),
// needed only when a per-key extends override is wanted.
func parseRequiredIdEntry(shapeName, key string, v any) RequiredIdEntry {
	if s, ok := v.(string); ok {
		return RequiredIdEntry{ID: s}
	}
	m, ok := v.(map[string]any)
	if !ok {
		failLoad("shapes.%s: requiredIds.%s must be a string or {\"value\":..., \"override\":...}", shapeName, key)
	}
	for k := range m {
		if k != "value" && k != "override" {
			failLoad("shapes.%s: requiredIds.%s: unknown key %q", shapeName, key, k)
		}
	}
	val, has := m["value"]
	if !has {
		failLoad("shapes.%s: requiredIds.%s needs a value", shapeName, key)
	}
	id, ok := val.(string)
	if !ok {
		failLoad("shapes.%s: requiredIds.%s.value must be a string", shapeName, key)
	}
	entry := RequiredIdEntry{ID: id}
	if ov, has := m["override"]; has {
		b, ok := ov.(bool)
		if !ok {
			failLoad("shapes.%s: requiredIds.%s.override must be a boolean", shapeName, key)
		}
		entry.Override = b
	}
	return entry
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
