// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the Apache License, Version 2.0.
// https://www.apache.org/licenses/LICENSE-2.0
package shaxon

// Phase 4.1 tests: index build, errors, ordering, reuse and cost
// (core sections 3, 4a and 5; decisions P1..P6 in indices.go).

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/ha1tch/jaxson/pkg/jaxson"
)

// val decodes any JSON value into the value model (decode is object-only).
func val(t *testing.T, src string) any {
	t.Helper()
	var raw any
	d := json.NewDecoder(strings.NewReader(src))
	d.UseNumber()
	if err := d.Decode(&raw); err != nil {
		t.Fatalf("invalid JSON fixture: %v", err)
	}
	return toJaxsonValue(raw)
}

func state0(t *testing.T, src string) map[string]any { return val(t, src).(map[string]any) }

func raise(f func()) (err *jaxson.Err) {
	defer func() {
		if r := recover(); r != nil {
			e, ok := r.(*jaxson.Err)
			if !ok {
				panic(r)
			}
			err = e
		}
	}()
	f()
	return nil
}

const pkgIdx = `{
  "indices": {
    "byId":    {"source": {"$path": ["state", "items"]}, "key": {"$path": ["local", "item", "id"]}},
    "byGroup": {"source": {"$path": ["state", "items"]}, "key": {"$path": ["local", "item", "g"]}, "multi": true},
    "byRate":  {"source": {"$path": ["state", "items"]}, "key": {"$compute": {"with": {"id": {"$path": ["local", "item", "id"]}, "r": {"$path": ["state", "rate"]}}, "expr": ["add", {"$v": "id"}, {"$v": "r"}]}}},
    "computed": {"source": {"$compute": {"with": {}, "expr": ["list", 1, 2]}}, "key": {"$path": ["local", "item"]}}
  },
  "relations": {
    "itemOwner": {"from": {"path": ["state", "items"], "field": "owner"}, "to": {"path": ["state", "owners"], "key": "id"}}
  },
  "shapes": {"Any": {"kind": "any"}}
}`

const stateIdx = `{
  "rate": 100,
  "other": 0,
  "items": [{"id": 3, "g": "b", "owner": "o2"}, {"id": 1, "g": "a", "owner": "o1"}, {"id": 2, "g": "b", "owner": "o1"}],
  "owners": [{"id": "o1"}, {"id": "o2"}]
}`

func newSet(t *testing.T, pkg, state string) (*IndexSet, *jaxson.Machine) {
	t.Helper()
	reg := parseReg(t, pkg)
	m := jaxson.NewMachine(nil, state0(t, state), nil, 100000, nil, nil)
	return NewIndexSet(m, reg), m
}

func TestIndexLookupAndKeyTypes(t *testing.T) {
	s, _ := newSet(t, pkgIdx, stateIdx)
	byId := ReferenceTarget{Index: "byId"}
	for _, c := range []struct {
		key  string
		want bool
	}{{"1", true}, {"2", true}, {"3", true}, {"4", false}, {"1.0", true}, {`"1"`, false}} {
		if got := s.Lookup(byId, val(t, c.key)); got != c.want {
			t.Errorf("Lookup(%s) = %v, want %v", c.key, got, c.want)
		}
	}
	for _, k := range []any{map[string]any{}, []any{}} {
		if err := raise(func() { s.Lookup(byId, k) }); err == nil || err.Code != "TYPE_ERROR" {
			t.Errorf("non-scalar lookup key %v: %v", k, err)
		}
	}
}

func TestIndexBuildErrors(t *testing.T) {
	cases := []struct {
		name, state, wantCat, wantCode string
	}{
		{"repeated key", `{"items": [{"id": 1}, {"id": 2}, {"id": 1}]}`, CatShapeError, ""},
		{"non-scalar key", `{"items": [{"id": {"x": 1}}]}`, "EXECUTION_ERROR", "TYPE_ERROR"},
		{"missing key", `{"items": [{"id": 1}, {"nope": 2}]}`, "EXECUTION_ERROR", "MISSING_PATH"},
		{"source not an array", `{"items": {"id": 1}}`, "EXECUTION_ERROR", "TYPE_ERROR"},
		{"source missing", `{}`, "EXECUTION_ERROR", "MISSING_PATH"},
	}
	for _, c := range cases {
		s, _ := newSet(t, pkgIdx, c.state)
		err := raise(func() { s.Lookup(ReferenceTarget{Index: "byId"}, val(t, "1")) })
		if err == nil || err.Cat != c.wantCat || err.Code != c.wantCode {
			t.Errorf("%s: %v, want %s/%s", c.name, err, c.wantCat, c.wantCode)
		}
	}
	// Repeats are fine when the index is multi.
	s, _ := newSet(t, pkgIdx, stateIdx)
	if !s.Lookup(ReferenceTarget{Index: "byGroup"}, "b") {
		t.Error("multi index lost a repeated key")
	}
	// The failed build left nothing half-built: a later call fails the same way.
	s, _ = newSet(t, pkgIdx, `{"items": [{"id": 1}, {"id": 1}]}`)
	for i := 0; i < 2; i++ {
		if err := raise(func() { s.Lookup(ReferenceTarget{Index: "byId"}, val(t, "1")) }); err == nil || err.Cat != CatShapeError {
			t.Errorf("attempt %d: %v", i, err)
		}
	}
}

func TestIndexedOrderAndGrouping(t *testing.T) {
	s, _ := newSet(t, pkgIdx, stateIdx)
	// Non-multi: ascending key order (ids 1, 2, 3 are at positions 1, 2, 0).
	want := [][]any{{"state", "items", 1}, {"state", "items", 2}, {"state", "items", 0}}
	if got := s.Elements("byId"); !reflect.DeepEqual(got, want) {
		t.Errorf("byId elements %v, want %v", got, want)
	}
	// Multi: grouped by key ascending (a, b), then by position within a key.
	want = [][]any{{"state", "items", 1}, {"state", "items", 0}, {"state", "items", 2}}
	if got := s.Elements("byGroup"); !reflect.DeepEqual(got, want) {
		t.Errorf("byGroup elements %v, want %v", got, want)
	}
	// Every source element appears exactly once.
	if n := len(s.Elements("byGroup")); n != 3 {
		t.Errorf("%d elements, want 3", n)
	}
}

func TestMixedKeyOrder(t *testing.T) {
	// P4: null < false < true < numbers < strings.
	pkg := `{"indices": {"k": {"source": {"$path": ["state", "xs"]}, "key": {"$path": ["local", "item"]}}}, "shapes": {"Any": {"kind": "any"}}}`
	s, _ := newSet(t, pkg, `{"xs": ["b", 10, true, null, "a", 2, false]}`)
	want := [][]any{{"state", "xs", 3}, {"state", "xs", 6}, {"state", "xs", 2}, {"state", "xs", 5}, {"state", "xs", 1}, {"state", "xs", 4}, {"state", "xs", 0}}
	if got := s.Elements("k"); !reflect.DeepEqual(got, want) {
		t.Errorf("order %v, want %v", got, want)
	}
}

func TestIndexBuildCostAndCharge(t *testing.T) {
	s, m := newSet(t, pkgIdx, stateIdx)
	byId := ReferenceTarget{Index: "byId"}
	s.Lookup(byId, "x")
	if m.Steps() != 3 {
		t.Errorf("build of 3 elements cost %d steps, want 3", m.Steps())
	}
	s.Lookup(byId, "y")
	s.Lookup(byId, "z")
	if m.Steps() != 3 {
		t.Errorf("reuse recharged: %d steps, want 3", m.Steps())
	}

	// The event is priced by the cost table.
	s, m = newSet(t, pkgIdx, stateIdx)
	tbl := jaxson.UnitTable()
	tbl.Event[EventIndexElement] = jaxson.Cost{Base: 4}
	if e := m.SetCostTable(tbl); e != nil {
		t.Fatal(e)
	}
	s.Lookup(byId, "x")
	if m.Steps() != 12 {
		t.Errorf("4 per element x 3 = %d, want 12", m.Steps())
	}

	// Charge before effect: a limit of 2 stops at the third element.
	reg := parseReg(t, pkgIdx)
	m2 := jaxson.NewMachine(nil, state0(t, stateIdx), nil, 2, nil, nil)
	err := raise(func() { NewIndexSet(m2, reg).Lookup(byId, "x") })
	if err == nil || err.Code != "STEPS" || m2.Steps() != 2 {
		t.Errorf("limit 2: err %v, steps %d", err, m2.Steps())
	}
}

// mutate runs one program against the machine, which fires OnMutate.
func mutate(t *testing.T, m *jaxson.Machine, prog string) {
	t.Helper()
	if err := raise(func() { m.RunProgram(val(t, prog).([]any)) }); err != nil {
		t.Fatalf("mutation %s: %v", prog, err)
	}
}

// Reuse (P1, P2): a build is recharged exactly when a mutation overlaps a
// path it depends on, and not otherwise.
func TestIndexReuseRule(t *testing.T) {
	cases := []struct {
		name, prog string
		target     ReferenceTarget
		recharge   int64 // steps expected for the next lookup
	}{
		{"same path under another root", `[{"op":"set","path":["output"],"value":{"items":[1]}}]`, ReferenceTarget{Index: "byId"}, 0},
		{"unrelated path", `[{"op":"set","path":["state","other"],"value":1}]`, ReferenceTarget{Index: "byId"}, 0},
		{"sibling of the source", `[{"op":"set","path":["state","owners"],"value":[]}]`, ReferenceTarget{Index: "byId"}, 0},
		{"inside an element (the source is a prefix of it)", `[{"op":"set","path":["state","items",1,"id"],"value":9}]`, ReferenceTarget{Index: "byId"}, 3},
		{"the source itself", `[{"op":"append","path":["state","items"],"value":{"id":7,"g":"c","owner":"o1"}}]`, ReferenceTarget{Index: "byId"}, 4},
		{"an ancestor of the source", `[{"op":"set","path":["state"],"value":{"items":[{"id":1,"g":"a","owner":"o1"}],"rate":0}}]`, ReferenceTarget{Index: "byId"}, 1},
		{"a path the key reads (P2)", `[{"op":"set","path":["state","rate"],"value":5}]`, ReferenceTarget{Index: "byRate"}, 6}, // 3 elements, plus the key's own add each
		{"a path the key does not read", `[{"op":"set","path":["state","rate"],"value":5}]`, ReferenceTarget{Index: "byId"}, 0},
		{"a relation's other side", `[{"op":"append","path":["state","owners"],"value":{"id":"o3"}}]`, ReferenceTarget{Relation: "itemOwner"}, 6},
	}
	for _, c := range cases {
		s, m := newSet(t, pkgIdx, stateIdx)
		s.Lookup(c.target, "x")
		before := m.Steps()
		mutate(t, m, c.prog)
		mutBefore := m.Steps()
		s.Lookup(c.target, "x")
		if got := m.Steps() - mutBefore; got != c.recharge {
			t.Errorf("%s: lookup after mutation cost %d, want %d (build cost %d)", c.name, got, c.recharge, before)
		}
	}
}

// A mutation that ran before the first build costs nothing and does not
// make a later build look stale; and the fresh data is what gets indexed.
func TestIndexSeesCurrentData(t *testing.T) {
	s, m := newSet(t, pkgIdx, stateIdx)
	byId := ReferenceTarget{Index: "byId"}
	if s.Lookup(byId, val(t, "9")) {
		t.Fatal("9 should be absent")
	}
	mutate(t, m, `[{"op":"append","path":["state","items"],"value":{"id":9,"g":"c","owner":"o1"}}]`)
	if !s.Lookup(byId, val(t, "9")) {
		t.Error("index served a stale snapshot after an append to its source")
	}
	mutate(t, m, `[{"op":"delete","path":["state","items",3]}]`)
	if s.Lookup(byId, val(t, "9")) {
		t.Error("index still holds an element deleted from its source")
	}
}

// NewIndexSet must not displace an OnMutate hook that was already set.
func TestIndexSetChainsOnMutate(t *testing.T) {
	reg := parseReg(t, pkgIdx)
	m := jaxson.NewMachine(nil, state0(t, stateIdx), nil, 1000, nil, nil)
	var seen []string
	m.OnMutate = func(root string, segs []any) { seen = append(seen, root) }
	NewIndexSet(m, reg)
	mutate(t, m, `[{"op":"set","path":["state","other"],"value":1}]`)
	if !reflect.DeepEqual(seen, []string{"state"}) {
		t.Errorf("earlier hook saw %v", seen)
	}
}

func TestRelationUnit(t *testing.T) {
	s, m := newSet(t, pkgIdx, stateIdx)
	rel := ReferenceTarget{Relation: "itemOwner"}
	if !s.Lookup(rel, "o1") || s.Lookup(rel, "o9") {
		t.Error("relation to-side lookup wrong")
	}
	// P3: both sides are built and charged: 2 owners + 3 items.
	if m.Steps() != 5 {
		t.Errorf("relation build cost %d, want 5", m.Steps())
	}
	// $inverse: items whose owner is o1, in array order.
	want := [][]any{{"state", "items", 1}, {"state", "items", 2}}
	if got := s.Inverse(rel, "o1"); !reflect.DeepEqual(got, want) {
		t.Errorf("inverse o1 = %v, want %v", got, want)
	}
	if got := s.Inverse(rel, "none"); len(got) != 0 {
		t.Errorf("inverse of an absent key = %v", got)
	}
	if m.Steps() != 5 {
		t.Errorf("inverse recharged: %d", m.Steps())
	}
}

func TestRelationOneToOneAndToSideUniqueness(t *testing.T) {
	one := strings.Replace(pkgIdx, `"to": {"path": ["state", "owners"], "key": "id"}}`, `"to": {"path": ["state", "owners"], "key": "id"}, "cardinality": "one-to-one"}`, 1)
	s, _ := newSet(t, one, stateIdx) // o1 owns two items
	err := raise(func() { s.Lookup(ReferenceTarget{Relation: "itemOwner"}, "o1") })
	if err == nil || err.Cat != CatShapeError || !strings.Contains(err.Msg, "one-to-one") {
		t.Errorf("one-to-one with a shared owner: %v", err)
	}
	// The default cardinality accepts the same data.
	s, _ = newSet(t, pkgIdx, stateIdx)
	if err := raise(func() { s.Lookup(ReferenceTarget{Relation: "itemOwner"}, "o1") }); err != nil {
		t.Errorf("one-to-many: %v", err)
	}
	// A repeated to.key is a SHAPE_ERROR whatever the cardinality.
	s, _ = newSet(t, pkgIdx, `{"items": [], "owners": [{"id": "a"}, {"id": "a"}]}`)
	if err := raise(func() { s.Lookup(ReferenceTarget{Relation: "itemOwner"}, "a") }); err == nil || err.Cat != CatShapeError {
		t.Errorf("repeated to.key: %v", err)
	}
}

func TestInlineShorthandAndPathlessSource(t *testing.T) {
	s, _ := newSet(t, pkgIdx, stateIdx)
	short := ReferenceTarget{Of: []any{map[string]any{"$path": []any{"state", "owners"}}}, By: []any{"id"}}
	if !s.Lookup(short, "o2") || s.Lookup(short, "o3") {
		t.Error("inline shorthand lookup wrong")
	}
	// P5: an index over a computed source serves lookups but has no paths.
	if !s.Lookup(ReferenceTarget{Index: "computed"}, val(t, "2")) {
		t.Error("computed-source lookup failed")
	}
	if err := raise(func() { s.Elements("computed") }); err == nil || err.Cat != CatShapeError {
		t.Errorf("Elements of a pathless index: %v", err)
	}
	// $inverse needs a multi index.
	if err := raise(func() { s.Inverse(ReferenceTarget{Index: "byId"}, val(t, "1")) }); err == nil || err.Cat != CatShapeError {
		t.Errorf("Inverse of a non-multi index: %v", err)
	}
}

// Through the evaluator: a reference-kind shape with no resolver given uses
// one IndexSet for the whole check, so several references share one build.
func TestEvaluatorSharesOneBuild(t *testing.T) {
	reg := parseReg(t, pkgRef)
	in := `{"tags": [{"id": "a"}, {"id": "b"}], "doc": {"owner": "a", "tagIds": ["a", "b", "a", "b"]}}`
	n := stepsUsed(t, reg, in, Options{}, "Doc", "doc")
	// Activations: Doc, tagIds field, 4 items (inline reference needs no
	// activation of its own beyond the item) are counted by Phase 3; the
	// index adds exactly its 2 elements, once.
	base := stepsUsed(t, reg, in, Options{Resolver: fakeResolver{"tagsById": {"a": true, "b": true}}}, "Doc", "doc")
	if n-base != 2 {
		t.Errorf("real index added %d steps over the fake, want 2", n-base)
	}
}
