// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the Apache License, Version 2.0.
// https://www.apache.org/licenses/LICENSE-2.0
package shaxon

// Phase 4.4 tests: the path-expression operand forms (core section 6;
// decisions F1..F7 in forms.go) and the jaxson operand-form hook behind them.

import (
	"strings"
	"testing"

	"github.com/ha1tch/jaxson/pkg/jaxson"
)

const pkgF = `{
  "indices": {
    "byOwner": {"source": {"$path": ["state","items"]}, "key": {"$path": ["local","item","owner"]}, "multi": true},
    "byId":    {"source": {"$path": ["state","items"]}, "key": {"$path": ["local","item","id"]}}
  },
  "relations": {
    "itemOwner": {"from": {"path": ["state","items"], "field": "owner"}, "to": {"path": ["state","owners"], "key": "id"}}
  },
  "shapes": {
    "Owner": {"kind": "object", "closed": false, "check": {
      "with": {"mine": {"$inverse": {"relation": "itemOwner", "key": {"$path": ["local","focus","id"]}}},
               "want": {"$path": ["local","focus","itemCount"]}},
      "expr": ["eq", ["len", {"$v": "mine"}], {"$v": "want"}]}},
    "Chain": {"kind": "object", "closed": false, "check": {
      "with": {"c": {"$path*": ["next"], "maxDepth": 5}, "want": {"$path": ["local","focus","expect"]}},
      "expr": ["eq", ["len", {"$v": "c"}], {"$v": "want"}]}},
    "Boss": {"kind": "object", "closed": false, "check": {
      "with": {"who": {"$altPath": [["local","focus","manager"], ["local","focus","lead"]]}},
      "expr": ["eq", {"$v": "who"}, "ann"]}}
  }
}`

const stateF = `{
  "owners": [{"id": "o1", "itemCount": 2}, {"id": "o2", "itemCount": 5}],
  "items": [{"id": 1, "owner": "o1"}, {"id": 2, "owner": "o2"}, {"id": 3, "owner": "o1"}],
  "c3": {"expect": 3, "next": {"next": {}}},
  "c1": {"expect": 1},
  "m": {"manager": "ann"},
  "l": {"lead": "ann"},
  "n": {"manager": "bob", "lead": "ann"},
  "chains": [{"expect": 1}, {"expect": 2, "next": {}}, {"expect": 3, "next": {"next": {}}}],
  "none": {}
}`

func runShape(t *testing.T, shape, path string) (Report, *jaxson.Err) {
	t.Helper()
	rt, _, reg := harness(t, pkgF, stateF, 100000)
	return rt.Run(entry(t, reg, `{"target": {"$path": `+path+`}, "shape": "`+shape+`", "mode": "report"}`), false)
}

func TestInverseInACheck(t *testing.T) {
	// o1 owns items 1 and 3; itemCount says 2: conforms.
	rep, err := runShape(t, "Owner", `["state","owners",0]`)
	if err != nil || !rep.Conforms() {
		t.Errorf("o1: %v, %v", rep.Violations, err)
	}
	// o2 owns one item but claims five.
	rep, err = runShape(t, "Owner", `["state","owners",1]`)
	if err != nil || rep.Conforms() || rep.Violations[0].Message != "check failed" {
		t.Errorf("o2: %v, %v", rep.Violations, err)
	}
}

func TestInverseValueIsPaths(t *testing.T) {
	rt, m, reg := harness(t, pkgF, stateF, 100000)
	prog := val(t, `[{"op":"set","path":["state","r"],"value":{"$inverse":{"index":"byOwner","key":"o1"}}},
	                 {"op":"set","path":["state","none"],"value":{"$inverse":{"relation":"itemOwner","key":"zz"}}}]`).([]any)
	if err := jaxson.CheckProgramForms(prog, rt.Instructions(), reg, rt.Forms()); err != nil {
		t.Fatal(err)
	}
	if err := raise(func() { m.RunProgram(prog) }); err != nil {
		t.Fatal(err)
	}
	want := val(t, `[["state","items",0],["state","items",2]]`)
	if got := m.GetAt("state", []any{"r"}); !jaxson.Equal(got, want) {
		t.Errorf("$inverse = %s, want %s", jaxson.Show(got), jaxson.Show(want))
	}
	if got := m.GetAt("state", []any{"none"}); !jaxson.Equal(got, val(t, `[]`)) {
		t.Errorf("an absent key should give an empty list, got %s", jaxson.Show(got))
	}
}

func TestAltPath(t *testing.T) {
	for _, c := range []struct {
		path string
		ok   bool
		code string
	}{
		{`["state","m"]`, true, ""},                 // first alternative
		{`["state","l"]`, true, ""},                 // only the second resolves
		{`["state","n"]`, false, ""},                // the first alternative that resolves wins, and it is "bob"
		{`["state","none"]`, false, "MISSING_PATH"}, // none resolve
	} {
		rep, err := runShape(t, "Boss", c.path)
		switch {
		case c.code != "":
			if err == nil || err.Code != c.code {
				t.Errorf("%s: err %v, want %s", c.path, err, c.code)
			}
		case err != nil:
			t.Errorf("%s: %v", c.path, err)
		case rep.Conforms() != c.ok:
			t.Errorf("%s: conforms=%v, want %v", c.path, rep.Conforms(), c.ok)
		}
	}
}

func TestClosureOperandFromAmbientPosition(t *testing.T) {
	// The closure starts at the focus node: c3 -> next -> next is 3 nodes.
	rep, err := runShape(t, "Chain", `["state","c3"]`)
	if err != nil || !rep.Conforms() {
		t.Errorf("c3: %v, %v", rep.Violations, err)
	}
	rep, err = runShape(t, "Chain", `["state","c1"]`)
	if err != nil || !rep.Conforms() {
		t.Errorf("c1 (one node): %v, %v", rep.Violations, err)
	}
	// Judged as an element of a collection, each element's own path is the base.
	rt, _, reg := harness(t, pkgF, stateF, 100000)
	rep, err = rt.Run(entry(t, reg, `{"target": {"$each": ["state","chains"]}, "shape": "Chain", "mode": "report"}`), false)
	if err != nil || !rep.Conforms() {
		t.Errorf("each chain from its own path: %v, %v", rep.Violations, err)
	}
}

func TestClosureOperandHorizonAndExplicitFrom(t *testing.T) {
	rt, m, reg := harness(t, pkgF, stateF, 100000)
	prog := val(t, `[
	  {"op":"set","path":["state","a"],"value":{"$path+":["next"],"from":["state","c3"],"maxDepth":5}},
	  {"op":"set","path":["state","b"],"value":{"$path*":["next"],"from":["state","c3"],"maxDepth":1}}]`).([]any)
	if err := jaxson.CheckProgramForms(prog, rt.Instructions(), reg, rt.Forms()); err != nil {
		t.Fatal(err)
	}
	err := raise(func() { m.RunProgram(prog) })
	// The first op works: two nodes beyond the base. The second hits the
	// horizon (F4).
	if got := m.GetAt("state", []any{"a"}).([]any); len(got) != 2 {
		t.Errorf("$path+ from c3 = %s", jaxson.Show(got))
	}
	if err == nil || err.Cat != "EXECUTION_ERROR" || err.Code != CodePathDepthExceeded {
		t.Errorf("horizon in an operand: %v", err)
	}
}

func TestFormsStaticRules(t *testing.T) {
	check := func(src string) *jaxson.Err {
		rt := NewRuntime(parseReg(t, pkgF), 0)
		return jaxson.CheckProgramForms(val(t, src).([]any), rt.Instructions(), rt.Reg, rt.Forms())
	}
	if err := check(`[{"op":"set","path":["state","x"],"value":{"$inverse":{"index":"byOwner","key":"o1"}}}]`); err != nil {
		t.Errorf("valid $inverse rejected: %v", err)
	}
	bad := map[string]string{
		"altPath empty":          `{"$altPath": []}`,
		"altPath nested":         `{"$altPath": [{"$altPath": [["state"]]}]}`,
		"altPath bad path":       `{"$altPath": [["nowhere"]]}`,
		"inverse both":           `{"$inverse": {"index": "byOwner", "relation": "itemOwner", "key": 1}}`,
		"inverse neither":        `{"$inverse": {"key": 1}}`,
		"inverse no key":         `{"$inverse": {"index": "byOwner"}}`,
		"inverse extra key":      `{"$inverse": {"index": "byOwner", "key": 1, "x": 1}}`,
		"inverse nested":         `{"$inverse": {"index": "byOwner", "key": {"$inverse": {"index": "byOwner", "key": 1}}}}`,
		"inverse extra form key": `{"$inverse": {"index": "byOwner", "key": 1}, "y": 1}`,
		"closure no maxDepth":    `{"$path*": ["next"], "from": ["state","c3"]}`,
		"closure zero maxDepth":  `{"$path*": ["next"], "from": ["state","c3"], "maxDepth": 0}`,
		"closure no ambient":     `{"$path*": ["next"], "maxDepth": 3}`,
		"closure empty step":     `{"$path*": [], "from": ["state","c3"], "maxDepth": 3}`,
		"closure unknown key":    `{"$path+": ["next"], "from": ["state","c3"], "maxDepth": 3, "z": 1}`,
		"closure from nested":    `{"$path*": ["next"], "from": {"$altPath": [["state"]]}, "maxDepth": 3}`,
	}
	for name, form := range bad {
		src := `[{"op":"set","path":["state","x"],"value":` + form + `}]`
		if err := check(src); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

func TestInverseNamesAreCheckedAtLoad(t *testing.T) {
	mk := func(shape string) string {
		return strings.Replace(pkgF, `"Boss":`, `"Extra": `+shape+`, "Boss":`, 1)
	}
	use := func(inv string) string {
		return `{"kind":"object","closed":false,"check":{"with":{"x":{"$inverse":` + inv + `}},"expr":["eq",1,1]}}`
	}
	for name, inv := range map[string]string{
		"undeclared index":    `{"index":"ghost","key":1}`,
		"undeclared relation": `{"relation":"ghost","key":1}`,
		"non-multi index":     `{"index":"byId","key":1}`,
	} {
		_, err := ParseRegistries(decode(t, mk(use(inv))))
		if err == nil || err.Cat != CatShapeError {
			t.Errorf("%s: %v", name, err)
		}
	}
	if _, err := ParseRegistries(decode(t, mk(use(`{"index":"byOwner","key":1}`)))); err != nil {
		t.Errorf("valid names rejected: %v", err)
	}
	// A computes entry is checked too, wherever it is referenced from.
	pkg := strings.Replace(pkgF, `"shapes": {`, `"computes": {"c": {"with": {"x": {"$inverse": {"relation": "ghost", "key": 1}}}, "expr": ["eq", 1, 1]}}, "shapes": {`, 1)
	if _, err := ParseRegistries(decode(t, pkg)); err == nil || !strings.Contains(err.Msg, "ghost") {
		t.Errorf("computes entry: %v", err)
	}
}

func TestInverseCyclesAreRefused(t *testing.T) {
	pkg := `{"indices": {"a": {"source": {"$path": ["state","xs"]}, "multi": true,
	  "key": {"$compute": {"with": {"r": {"$inverse": {"index": "a", "key": 1}}}, "expr": ["len", {"$v": "r"}]}}}},
	  "shapes": {"Any": {"kind": "any"}}}`
	rt, _, _ := harness(t, pkg, `{"xs": [1]}`, 1000)
	err := raise(func() { rt.idx.Elements("a") })
	if err == nil || err.Cat != CatShapeError || !strings.Contains(err.Msg, "F7") {
		t.Errorf("self-referential index: %v", err)
	}
	// The failed build did not wedge the set: asking again fails the same way, not by recursion.
	if err2 := raise(func() { rt.idx.Elements("a") }); err2 == nil || err2.Cat != CatShapeError {
		t.Errorf("second attempt: %v", err2)
	}
}

func TestIndexKeyMayUseAnAmbientClosure(t *testing.T) {
	// The key of an index is evaluated per element with that element's path
	// as the ambient position: a closure over each tree node gives its depth.
	pkg := `{"indices": {"byDepth": {"source": {"$path": ["state","nodes"]}, "multi": true,
	  "key": {"$compute": {"with": {"c": {"$path*": ["next"], "maxDepth": 4}}, "expr": ["len", {"$v": "c"}]}}}},
	  "shapes": {"Any": {"kind": "any"}}}`
	rt, _, _ := harness(t, pkg, `{"nodes": [{"next": {}}, {}, {"next": {"next": {}}}]}`, 1000)
	if !rt.idx.Lookup(ReferenceTarget{Index: "byDepth"}, val(t, "2")) || !rt.idx.Lookup(ReferenceTarget{Index: "byDepth"}, val(t, "1")) || !rt.idx.Lookup(ReferenceTarget{Index: "byDepth"}, val(t, "3")) {
		t.Error("closure lengths 1, 2 and 3 should all be keys")
	}
}

func TestInverseRefreshedBeforeTargetsAndChargedOnce(t *testing.T) {
	rt, m, reg := harness(t, pkgF, stateF, 100000)
	// V7: a missing target still leaves the relation's build charged (2+3).
	_, err := rt.Run(entry(t, reg, `{"target": {"$path": ["state","ghost"]}, "shape": "Owner", "mode": "gate"}`), false)
	if err == nil || err.Code != "MISSING_PATH" || m.Steps() != 5 {
		t.Errorf("err %v, steps %d; want MISSING_PATH after 5 build steps", err, m.Steps())
	}
}

func TestEvaluatorOnABareMachineGetsTheForms(t *testing.T) {
	reg := parseReg(t, pkgF)
	m := jaxson.NewMachine(nil, state0(t, stateF), nil, 100000, nil, nil)
	ev := NewEvaluator(m, reg, Options{})
	rep, err := ev.Check("Owner", "state", []any{"owners", 0})
	if err != nil || !rep.Conforms() {
		t.Errorf("bare machine: %v, %v", rep.Violations, err)
	}
}

// The hook is the core's only change: with no forms registered, a core
// program treats `$inverse` as the unknown form it always was.
func TestCoreRejectsShaxonFormsWithoutTheHook(t *testing.T) {
	err := jaxson.CheckProgram(val(t, `[{"op":"set","path":["state","x"],"value":{"$inverse":{"index":"i","key":1}}}]`).([]any), jaxson.CoreInstructions())
	if err == nil || !strings.Contains(err.Msg, "unknown form") {
		t.Errorf("core: %v", err)
	}
}
