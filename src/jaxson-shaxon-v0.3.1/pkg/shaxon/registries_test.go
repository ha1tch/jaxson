// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the GNU General Public License, version 3.
// https://www.gnu.org/licenses/gpl-3.0.html
package shaxon

import (
	"encoding/json"
	"math/big"
	"strings"
	"testing"
)

// decode parses a JSON literal into the map[string]any shape jaxson
// itself expects, with *big.Rat for every number — go's encoding/json
// decodes numbers as float64 by default, which the rest of this
// package's type assertions (v.(*big.Rat)) would reject, so this helper
// walks the result and converts.
func decode(t *testing.T, src string) map[string]any {
	t.Helper()
	var raw any
	dec := json.NewDecoder(strings.NewReader(src))
	dec.UseNumber()
	if err := dec.Decode(&raw); err != nil {
		t.Fatalf("invalid JSON fixture: %v", err)
	}
	return toJaxsonValue(raw).(map[string]any)
}

func toJaxsonValue(v any) any {
	switch t := v.(type) {
	case json.Number:
		r, ok := new(big.Rat).SetString(t.String())
		if !ok {
			panic("bad number literal in fixture: " + t.String())
		}
		return r
	case map[string]any:
		out := map[string]any{}
		for k, e := range t {
			out[k] = toJaxsonValue(e)
		}
		return out
	case []any:
		out := make([]any, len(t))
		for i, e := range t {
			out[i] = toJaxsonValue(e)
		}
		return out
	default:
		return v
	}
}

func TestParseRegistries_ValidPackage(t *testing.T) {
	pkg := decode(t, `{
		"shapes": {
			"Order": {
				"kind": "object",
				"closed": true,
				"fields": {
					"id":     {"kind": "string", "minLen": 1},
					"status": {"kind": "string", "enum": ["draft", "placed", "shipped"]},
					"total":  {"kind": "number", "min": 0}
				},
				"required": ["id", "status"],
				"requiredIds": {"id": "ORDER_NEEDS_ID"},
				"check": {
					"with": {"t": {"$path": ["local", "focus", "total"]}},
					"expr": ["ge", {"$v": "t"}, 0],
					"id": "ORDER_TOTAL_NONNEGATIVE"
				},
				"severity": "violation"
			},
			"LineItem": {
				"kind": "object",
				"fields": {"sku": {"kind": "string"}, "qty": {"kind": "number", "int": true, "min": 1}},
				"required": ["sku", "qty"]
			}
		},
		"computes": {
			"nonNegativeTotal": {
				"with": {"x": {"$path": ["local", "focus", "total"]}},
				"expr": ["ge", {"$v": "x"}, 0]
			}
		},
		"indices": {
			"ordersById": {
				"source": {"$path": ["input", "orders"]},
				"key": {"$path": ["local", "item", "id"]}
			}
		},
		"relations": {
			"orderCustomer": {
				"from": {"path": ["input", "orders"], "field": "customerId"},
				"to":   {"path": ["input", "customers"], "key": "id"},
				"cardinality": "one-to-many"
			}
		}
	}`)
	r, err := ParseRegistries(pkg)
	if err != nil {
		t.Fatalf("expected a clean parse, got %v", err)
	}
	if len(r.Shapes) != 2 {
		t.Errorf("expected 2 shapes, got %d", len(r.Shapes))
	}
	if len(r.Indices) != 1 || len(r.Relations) != 1 || len(r.Computes) != 1 {
		t.Errorf("expected 1 each of indices/relations/computes, got %d/%d/%d",
			len(r.Indices), len(r.Relations), len(r.Computes))
	}
}

func TestParseRegistries_ExtendsMerge(t *testing.T) {
	pkg := decode(t, `{
		"shapes": {
			"Base": {
				"kind": "object",
				"closed": true,
				"fields": {"id": {"kind": "string"}},
				"required": ["id"]
			},
			"Order": {
				"extends": "Base",
				"fields": {"total": {"kind": "number"}},
				"required": ["total"]
			}
		}
	}`)
	r, err := ParseRegistries(pkg)
	if err != nil {
		t.Fatalf("expected a clean parse, got %v", err)
	}
	order := r.Shapes["Order"]
	if len(order.Fields) != 2 {
		t.Errorf("expected Order's merged fields to include both id and total, got %d fields", len(order.Fields))
	}
	if len(order.Required) != 2 {
		t.Errorf("expected Order's merged required to union to 2 entries, got %d", len(order.Required))
	}
	if !order.Closed {
		t.Errorf("expected Order.Closed to be true (AND of two true values)")
	}
}

func wantShapeError(t *testing.T, pkg map[string]any, substr string) {
	t.Helper()
	_, err := ParseRegistries(pkg)
	if err == nil {
		t.Fatalf("expected a SHAX_SHAPE_ERROR, got a clean parse")
	}
	if err.Cat != CatShapeError {
		t.Errorf("expected category %s, got %s", CatShapeError, err.Cat)
	}
	if substr != "" && !strings.Contains(err.Msg, substr) {
		t.Errorf("expected message to contain %q, got %q", substr, err.Msg)
	}
}

func TestParseRegistries_ExtendsCycle(t *testing.T) {
	pkg := decode(t, `{
		"shapes": {
			"A": {"extends": "B", "fields": {"x": {"kind": "string"}}},
			"B": {"extends": "A", "fields": {"y": {"kind": "string"}}}
		}
	}`)
	wantShapeError(t, pkg, "cycle")
}

func TestParseRegistries_FieldCollisionOnExtends(t *testing.T) {
	pkg := decode(t, `{
		"shapes": {
			"Base": {"kind": "object", "fields": {"id": {"kind": "string"}}},
			"Child": {"extends": "Base", "fields": {"id": {"kind": "number"}}}
		}
	}`)
	wantShapeError(t, pkg, "collides")
}

func TestParseRegistries_RequiredIdsNotInRequired(t *testing.T) {
	pkg := decode(t, `{
		"shapes": {
			"Order": {
				"kind": "object",
				"fields": {"id": {"kind": "string"}},
				"requiredIds": {"id": "SOME_ID"}
			}
		}
	}`)
	wantShapeError(t, pkg, "not in required")
}

func TestParseRegistries_EnumOnDisallowedKind(t *testing.T) {
	pkg := decode(t, `{
		"shapes": {
			"Bad": {"kind": "object", "enum": [{"a": 1}]}
		}
	}`)
	wantShapeError(t, pkg, "not legal")
}

func TestParseRegistries_UnknownKind(t *testing.T) {
	pkg := decode(t, `{"shapes": {"Bad": {"kind": "integer"}}}`)
	wantShapeError(t, pkg, "unknown kind")
}

func TestParseRegistries_ReferenceNeedsExactlyOneForm(t *testing.T) {
	pkg := decode(t, `{
		"shapes": {
			"Bad": {"kind": "reference"}
		}
	}`)
	wantShapeError(t, pkg, "exactly one")
}

func TestParseRegistries_ReferenceBothIndexAndRelation(t *testing.T) {
	pkg := decode(t, `{
		"indices": {
			"idx": {"source": {"$path": ["input", "xs"]}, "key": {"$path": ["local", "item", "id"]}}
		},
		"relations": {
			"rel": {
				"from": {"path": ["input", "ys"], "field": "xId"},
				"to":   {"path": ["input", "xs"], "key": "id"}
			}
		},
		"shapes": {
			"Bad": {"kind": "reference", "index": "idx", "relation": "rel"}
		}
	}`)
	wantShapeError(t, pkg, "exactly one")
}

func TestParseRegistries_ComputeBadArity(t *testing.T) {
	pkg := decode(t, `{
		"computes": {
			"bad": {"with": {}, "expr": ["not", true, false]}
		}
	}`)
	wantShapeError(t, pkg, "")
}

func TestParseRegistries_ComputeUndeclaredV(t *testing.T) {
	pkg := decode(t, `{
		"computes": {
			"bad": {"with": {"x": true}, "expr": {"$v": "y"}}
		}
	}`)
	wantShapeError(t, pkg, "")
}

func TestParseRegistries_CheckReferencesUndeclaredCompute(t *testing.T) {
	pkg := decode(t, `{
		"shapes": {
			"Order": {"kind": "object", "check": {"compute": "doesNotExist"}}
		}
	}`)
	wantShapeError(t, pkg, "undeclared computes")
}

func TestParseRegistries_RelationMalformedCardinality(t *testing.T) {
	pkg := decode(t, `{
		"relations": {
			"rel": {
				"from": {"path": ["input", "ys"], "field": "xId"},
				"to":   {"path": ["input", "xs"], "key": "id"},
				"cardinality": "many-to-many"
			}
		}
	}`)
	wantShapeError(t, pkg, "cardinality")
}

func TestParseRegistries_RelationsRedundancy(t *testing.T) {
	pkg := decode(t, `{
		"relations": {
			"a": {"from": {"path": ["input", "ys"], "field": "xId"}, "to": {"path": ["input", "xs"], "key": "id"}},
			"b": {"from": {"path": ["input", "zs"], "field": "xId"}, "to": {"path": ["input", "xs"], "key": "id"}}
		}
	}`)
	wantShapeError(t, pkg, "same (path, key)")
}

func TestParseRegistries_RecursionNeedsMaxShapeDepth(t *testing.T) {
	pkg := decode(t, `{
		"shapes": {
			"Category": {
				"kind": "object",
				"fields": {"children": {"kind": "array", "items": {"shape": "Category"}}}
			}
		}
	}`)
	wantShapeError(t, pkg, "maxShapeDepth")
}

func TestParseRegistries_RecursionWithMaxShapeDepthOK(t *testing.T) {
	pkg := decode(t, `{
		"limits": {"maxShapeDepth": 10},
		"shapes": {
			"Category": {
				"kind": "object",
				"fields": {"children": {"kind": "array", "items": {"shape": "Category"}}}
			}
		}
	}`)
	if _, err := ParseRegistries(pkg); err != nil {
		t.Fatalf("expected a clean parse with maxShapeDepth declared, got %v", err)
	}
}

func TestParseRegistries_KeywordWrongKind(t *testing.T) {
	pkg := decode(t, `{"shapes": {"Bad": {"kind": "boolean", "minLen": 5}}}`)
	wantShapeError(t, pkg, "not legal on kind")
}

func TestParseRegistries_MinLenExceedsMaxLen(t *testing.T) {
	pkg := decode(t, `{"shapes": {"Bad": {"kind": "string", "minLen": 10, "maxLen": 2}}}`)
	wantShapeError(t, pkg, "exceeds maxLen")
}

func TestParseRegistries_MinExceedsMax(t *testing.T) {
	pkg := decode(t, `{"shapes": {"Bad": {"kind": "number", "min": 10, "max": 2}}}`)
	wantShapeError(t, pkg, "exceeds max")
}

func TestParseRegistries_MinItemsExceedsMaxItems(t *testing.T) {
	pkg := decode(t, `{"shapes": {"Bad": {"kind": "array", "items": {"kind": "string"}, "minItems": 5, "maxItems": 1}}}`)
	wantShapeError(t, pkg, "exceeds maxItems")
}

// TestParseRegistries_ExtendsInheritsKindForKeywordCheck is the case that
// motivated moving keyword-vs-kind validation to run after extends
// resolves: StrBase declares kind but no length bound; Order extends it
// and adds minLen without repeating kind. This must parse cleanly —
// validating minLen against Kind before the merge would wrongly reject it
// (Order's own pre-merge Kind is "").
func TestParseRegistries_ExtendsInheritsKindForKeywordCheck(t *testing.T) {
	pkg := decode(t, `{
		"shapes": {
			"StrBase": {"kind": "string"},
			"Padded":  {"extends": "StrBase", "minLen": 3}
		}
	}`)
	r, err := ParseRegistries(pkg)
	if err != nil {
		t.Fatalf("expected a clean parse (minLen legal once Kind inherits as string), got %v", err)
	}
	if r.Shapes["Padded"].Kind != "string" {
		t.Errorf("expected Padded.Kind to inherit \"string\" from StrBase, got %q", r.Shapes["Padded"].Kind)
	}
}

func TestParseRegistries_InlineShapeCannotExtend(t *testing.T) {
	pkg := decode(t, `{
		"shapes": {
			"Base": {"kind": "string"},
			"Order": {"kind": "object", "fields": {"id": {"extends": "Base", "kind": "string"}}}
		}
	}`)
	wantShapeError(t, pkg, "only valid on a named, top-level shape")
}

func TestParseRegistries_PerFieldOverride(t *testing.T) {
	pkg := decode(t, `{
		"shapes": {
			"Base": {"kind": "object", "fields": {"legacyCode": {"kind": "number"}}},
			"Child": {
				"extends": "Base",
				"fields": {"legacyCode": {"kind": "string", "override": true}}
			}
		}
	}`)
	r, err := ParseRegistries(pkg)
	if err != nil {
		t.Fatalf("expected a clean parse with override:true, got %v", err)
	}
	fd := r.Shapes["Child"].Fields["legacyCode"]
	if fd.Inline == nil || fd.Inline.Kind != "string" {
		t.Errorf("expected the child's override to win (kind=string), got %+v", fd)
	}
}

func TestParseRegistries_FieldCollisionWithoutOverrideStillErrors(t *testing.T) {
	// Same shapes as the override test above, minus "override": true —
	// must still be the ordinary collision error, proving override is
	// opt-in, not a silent behavior change to the default.
	pkg := decode(t, `{
		"shapes": {
			"Base": {"kind": "object", "fields": {"legacyCode": {"kind": "number"}}},
			"Child": {"extends": "Base", "fields": {"legacyCode": {"kind": "string"}}}
		}
	}`)
	wantShapeError(t, pkg, "collides")
}

func TestParseRegistries_OverrideRejectedOutsideFields(t *testing.T) {
	// "override" has no meaning on an items/and/or/xone/not position —
	// only fooOverride sibling keys do there. Must be rejected, not
	// silently accepted and ignored.
	pkg := decode(t, `{
		"shapes": {
			"Bad": {"kind": "array", "items": {"kind": "string", "override": true}}
		}
	}`)
	wantShapeError(t, pkg, "only valid inside a fields entry")
}

func TestParseRegistries_ClosedOverride(t *testing.T) {
	pkg := decode(t, `{
		"shapes": {
			"Base": {"kind": "object", "closed": true, "fields": {"id": {"kind": "string"}}},
			"Child": {"extends": "Base", "closed": false, "closedOverride": true}
		}
	}`)
	r, err := ParseRegistries(pkg)
	if err != nil {
		t.Fatalf("expected a clean parse, got %v", err)
	}
	if r.Shapes["Child"].Closed {
		t.Errorf("expected closedOverride:true to make Child.Closed false (its own value, not AND with parent's true)")
	}
}

func TestParseRegistries_AndOverrideReplacesRatherThanConcatenates(t *testing.T) {
	pkg := decode(t, `{
		"shapes": {
			"Base": {"kind": "object", "and": [{"kind": "object"}]},
			"Child": {
				"extends": "Base",
				"andOverride": true,
				"and": [{"kind": "object", "required": ["x"]}]
			}
		}
	}`)
	r, err := ParseRegistries(pkg)
	if err != nil {
		t.Fatalf("expected a clean parse, got %v", err)
	}
	if len(r.Shapes["Child"].And) != 1 {
		t.Errorf("expected andOverride:true to replace (length 1), not concatenate (length 2); got %d", len(r.Shapes["Child"].And))
	}
}

func TestParseRegistries_CheckOverrideReplacesRatherThanAnds(t *testing.T) {
	pkg := decode(t, `{
		"shapes": {
			"Base": {
				"kind": "object",
				"check": {"with": {"x": {"$path": ["local", "focus"]}}, "expr": ["eq", 1, 1]}
			},
			"Child": {
				"extends": "Base",
				"checkOverride": {"with": {"y": {"$path": ["local", "focus"]}}, "expr": ["eq", 2, 2]}
			}
		}
	}`)
	r, err := ParseRegistries(pkg)
	if err != nil {
		t.Fatalf("expected a clean parse, got %v", err)
	}
	checks := r.Shapes["Child"].Check
	if len(checks) != 1 {
		t.Fatalf("expected checkOverride to replace (1 check), not AND (2 checks); got %d", len(checks))
	}
	if checks[0].With["y"] == nil {
		t.Errorf("expected the surviving check to be the override's own (with key y), got %+v", checks[0].With)
	}
}

func TestParseRegistries_CheckAndCheckOverrideMutuallyExclusive(t *testing.T) {
	pkg := decode(t, `{
		"shapes": {
			"Base": {"kind": "object"},
			"Child": {
				"extends": "Base",
				"check": {"expr": true},
				"checkOverride": {"expr": false}
			}
		}
	}`)
	wantShapeError(t, pkg, "cannot both be declared")
}

func TestParseRegistries_RequiredIdPerKeyOverride(t *testing.T) {
	pkg := decode(t, `{
		"shapes": {
			"Base": {
				"kind": "object", "fields": {"id": {"kind": "string"}},
				"required": ["id"], "requiredIds": {"id": "BASE_NEEDS_ID"}
			},
			"Child": {
				"extends": "Base",
				"requiredIds": {"id": {"value": "CHILD_NEEDS_ID", "override": true}}
			}
		}
	}`)
	r, err := ParseRegistries(pkg)
	if err != nil {
		t.Fatalf("expected a clean parse, got %v", err)
	}
	if got := r.Shapes["Child"].RequiredIds["id"].ID; got != "CHILD_NEEDS_ID" {
		t.Errorf("expected the child's override to win, got %q", got)
	}
}

func TestParseRegistries_RequiredIdCollisionWithoutOverrideErrors(t *testing.T) {
	pkg := decode(t, `{
		"shapes": {
			"Base": {
				"kind": "object", "fields": {"id": {"kind": "string"}},
				"required": ["id"], "requiredIds": {"id": "BASE_NEEDS_ID"}
			},
			"Child": {
				"extends": "Base",
				"requiredIds": {"id": "CHILD_NEEDS_ID"}
			}
		}
	}`)
	wantShapeError(t, pkg, "collides")
}

func TestParseRegistries_ShorthandRedundantWithRelation(t *testing.T) {
	pkg := decode(t, `{
		"relations": {
			"orderCustomer": {
				"from": {"path": ["input", "orders"], "field": "customerId"},
				"to":   {"path": ["input", "customers"], "key": "id"}
			}
		},
		"shapes": {
			"Order": {
				"kind": "object",
				"fields": {
					"customerId": {"kind": "reference", "of": {"$path": ["input", "customers"]}, "by": ["id"]}
				}
			}
		}
	}`)
	wantShapeError(t, pkg, "duplicates")
}
