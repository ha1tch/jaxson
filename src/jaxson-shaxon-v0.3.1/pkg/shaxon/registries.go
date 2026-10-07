// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the GNU General Public License, version 3.
// https://www.gnu.org/licenses/gpl-3.0.html
package shaxon

// Phase 2 (jaxson-shaxon-go-implementation-plan.md section 5): parse
// indices/relations/shapes/computes once, at load time, independent of
// any input — the load-time half of what Phase 5's Hooks.Static will
// call. No Machine runs here and no EXECUTION_ERROR/VALIDATION_ERROR is
// possible from this file; only SHAX_SHAPE_ERROR.
//
// Scope note: primitive keywords a "string"/"number"/"array" kind may
// carry (minLen/maxLen, min/max/int, minItems/maxItems) are passed
// through in KeywordDecl and checked for the one rule core section 4
// states explicitly (enum restricted to string/number/boolean/null).
// Cross-keyword sanity (e.g. minLen <= maxLen) is not yet enforced —
// deliberately deferred, not overlooked; flagged again at the end of
// checkField below.

import (
	"math/big"

	"github.com/ha1tch/jaxson/pkg/jaxson"
)

// ---------------------------------------------------------------- declared vocabulary

var validKinds = map[string]bool{
	"object": true, "array": true, "string": true, "number": true,
	"boolean": true, "null": true, "any": true, "reference": true, "node": true,
}

// enumKinds: core section 4 — "enum is legal on string, number, boolean,
// and null kinds ... deliberately not extended to object/array."
var enumKinds = map[string]bool{"string": true, "number": true, "boolean": true, "null": true}

var validSeverities = map[string]bool{"violation": true, "warning": true, "info": true}

// ---------------------------------------------------------------- declared types

// FieldDecl is a shape used in field/items/qualified/not/combinator
// position: either a named reference ({"shape": "Name"}) or an inline
// shape declaration. Exactly one of ShapeRef/Inline is set.
// FieldDecl's Override is meaningful only when this FieldDecl is a value
// in some shape's Fields map (core section 4: "per-field override:true
// replaces the parent's field definition"); ignored elsewhere since
// and/or/xone/not/items/qualified override only at the whole-member
// level (ShapeDecl's AndOverride etc.) — parseField's allowOverride
// parameter rejects "override" as an unknown key anywhere else, rather
// than silently accepting a flag that would never be consulted.
type FieldDecl struct {
	ShapeRef string
	Inline   *ShapeDecl
	Override bool
}

// QualifiedDecl is core section 4's "between Min and Max of this
// collection's elements match Target."
type QualifiedDecl struct {
	Target   FieldDecl
	Min, Max *int64
}

// RequiredIdEntry is one requiredIds value. A bare JSON string is sugar
// for {ID: thatString, Override: false} — promoted to the object form
// only when a per-key override is needed, the same move RDF/SHACL make
// whenever a literal needs metadata attached to it.
type RequiredIdEntry struct {
	ID       string
	Override bool
}

// CombinatorCheck is one `check` — either authored inline (With/Expr set,
// ComputeRef empty) or as a reference into the package's computes
// registry (ComputeRef set, With/Expr nil). A shape accumulates more than
// one only via extends (core section 4: "check | parent's and child's
// check are ANDed").
type CombinatorCheck struct {
	With       map[string]any
	Expr       any
	ComputeRef string
	ID         string // constraintId; "" if not declared
}

// KeywordDecl carries a string/number/array kind's primitive keywords
// through, unevaluated — Phase 3 applies them to real data with
// jaxson.CheckStringLen/CheckNumber/CheckArrayLen, the same functions
// Jaxson's own inputSchema validation uses.
type KeywordDecl struct {
	MinLen, MaxLen     *int64
	Enum               []any
	Min, Max           *float64 // structural sanity only (Phase 2); inexact for decimals, never used to judge data
	MinRat, MaxRat     *big.Rat // the exact bounds, set alongside Min/Max; evaluation (Phase 3) reads these
	WantInt            bool
	MinItems, MaxItems *int64
}

// ShapeDecl is one shape, after extends has been fully resolved — every
// field here reflects the merged, final shape (plan section 5: "the
// extends merge table ... is implemented as one general merge-shape
// function," run once at load time, not at evaluation time).
//
// Not and Check are modelled as slices even though a single, non-extended
// shape has at most one entry: core section 4's merge rule says "parent's
// and child's not targets both apply" and "parent's and child's check are
// ANDed" — both become a conjunction of independently-satisfied
// constraints once extends accumulates them, which a slice represents
// directly without inventing a synthetic wrapper shape.
type ShapeDecl struct {
	Name string
	Kind string // "" is legal: "a shape using only combinators and no
	// kind/fields is a node-kind shape" (core section 4)
	Closed            bool
	ClosedSet         bool // the shape wrote "closed" itself (extends merge: an unstated value inherits)
	IgnoredProperties []string
	Fields            map[string]FieldDecl
	Required          []string
	RequiredIds       map[string]RequiredIdEntry
	And, Or, Xone     []FieldDecl
	Not               []FieldDecl
	Items             *FieldDecl // set when Kind == "array" and "items" is declared
	Qualified         map[string]QualifiedDecl
	Check             []CombinatorCheck
	Severity          string
	Message           string
	ID                string // constraintId, from an "id" member; "" if not declared
	Keywords          KeywordDecl

	// Whole-member extends-override flags (core section 4's merge table).
	// Fields and RequiredIds override per-item instead — see FieldDecl.
	// Override and RequiredIdEntry.Override. Required has no override: the
	// table states none is needed, since union only ever gets stricter.
	ClosedOverride            bool
	AndOverride, OrOverride   bool
	XoneOverride, NotOverride bool
	CheckOverride             *CombinatorCheck

	// Reference-kind members (core section 5); at most one of Index,
	// Relation, (Of+By) is set — parseField enforces exactly one.
	RefIndex     string
	RefRelation  string
	RefOf, RefBy []any
}

type IndexDecl struct {
	Name   string
	Source any // an operand ({"$path": [...]}, typically) resolving to an array; Phase 4 evaluates it
	Key    any // an operand evaluated per element with "item" bound; must resolve to a scalar (Phase 4)
	Multi  bool
}

// RelationDecl mirrors core section 4a exactly: from/to are each a literal
// path plus a plain field name, not operands — {"path": [...], "field"/
// "key": "name"}. Cardinality is "one-to-many" (default) or "one-to-one".
type RelationDecl struct {
	Name        string
	FromPath    []any
	FromField   string
	ToPath      []any
	ToKey       string
	Cardinality string
}

type ComputeDecl struct {
	Name string
	With map[string]any
	Expr any
}

// Registries is every package-level declaration, parsed and statically
// checked, ready for Phase 3/4 to evaluate against real data.
type Registries struct {
	Shapes    map[string]ShapeDecl
	Indices   map[string]IndexDecl
	Relations map[string]RelationDecl
	Computes  map[string]ComputeDecl
}

// ---------------------------------------------------------------- entry point

// ParseRegistries parses and statically validates a package's shapes,
// indices, relations, and computes. It does not look at validate,
// program, or input — those belong to later phases and later pipeline
// stages (core section 9: "parse, version, schemas and shapes and
// indices and relations (all static)" precedes everything else).
func ParseRegistries(pkg map[string]any) (r *Registries, err *jaxson.Err) {
	defer func() {
		if rec := recover(); rec != nil {
			e, ok := rec.(*jaxson.Err)
			if !ok {
				panic(rec)
			}
			err = e
		}
	}()
	r = &Registries{
		Shapes:    map[string]ShapeDecl{},
		Indices:   map[string]IndexDecl{},
		Relations: map[string]RelationDecl{},
		Computes:  map[string]ComputeDecl{},
	}
	parseComputes(pkg, r)  // must run before shapes: check.compute references this
	parseIndices(pkg, r)   // must run before relations: 4c compares against declared indices too
	parseRelations(pkg, r) // must run before shapes: reference.relation names this
	parseShapes(pkg, r)
	checkInverseNames(r) // needs everything above: a check may name a relation parsed after it
	checkRecursionBound(pkg, r)
	return r, nil
}

// wrapCompute turns a with/expr pair into the $compute operand form
// jaxson.CheckOperand expects, so Shaxon delegates arity/operator/$v
// checking to Jaxson's own machinery rather than duplicating it.
func wrapCompute(with map[string]any, expr any) map[string]any {
	body := map[string]any{"expr": expr}
	if with != nil {
		body["with"] = with
	}
	return map[string]any{"$compute": body}
}

// checkComputeBody validates a with/expr pair exactly as an inline
// $compute would be validated inside a program — "focus" is the one
// local every check/computes body may reference (core section 4/4d).
func checkComputeBody(with map[string]any, expr any, context string) {
	if expr == nil {
		failLoad("%s: needs an expr", context)
	}
	if e := jaxson.CheckOperandWith(StaticForms(), wrapCompute(with, expr), "focus"); e != nil {
		failLoad("%s: %s", context, e.Msg)
	}
}
