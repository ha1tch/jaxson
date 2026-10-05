// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the Apache License, Version 2.0.
// https://www.apache.org/licenses/LICENSE-2.0
package shaxon

// Phase 4.3 tests: bounded path closures as targets (core section 6;
// decisions C1..C6 in paths.go).

import (
	"reflect"
	"testing"

	"github.com/ha1tch/jaxson/pkg/jaxson"
)

const pkgList = `{
  "shapes": {"Node": {"kind": "object", "closed": false, "required": ["v"], "fields": {"v": {"kind": "number"}}}}
}`

const stateList = `{
  "list": {"v": 1, "next": {"v": 2, "next": {"v": 3, "next": {"v": 4}}}},
  "walk": {"dir": "b", "a": {"v": 9}, "b": {"dir": "a", "a": {"dir": "x", "x": {"v": "end"}}}},
  "grid": [[[1], 2], 3],
  "bad":  {"dir": [1], "a": {"v": 1}},
  "lone": {"v": 1}
}`

func closureOf(t *testing.T, state, target string) (paths [][]any, steps int64, depthAt []any, err *jaxson.Err) {
	t.Helper()
	rt, m, reg := harness(t, pkgList, state, 100000)
	tg := parseTarget("t", val(t, target), reg)
	var foci []Focus
	err = raise(func() { foci, depthAt = tg.resolve(m, rt.idx) })
	return pathsOf(foci), m.Steps(), depthAt, err
}

func pathsOf(f []Focus) [][]any {
	out := make([][]any, len(f))
	for i, x := range f {
		out[i] = x.Path
	}
	return out
}

func TestClosureStarAndPlus(t *testing.T) {
	p := func(segs ...any) []any { return segs }
	star := `{"$path*": ["next"], "from": ["state","list"], "maxDepth": 10}`
	got, steps, depthAt, err := closureOf(t, stateList, star)
	want := [][]any{p("state", "list"), p("state", "list", "next"), p("state", "list", "next", "next"), p("state", "list", "next", "next", "next")}
	if err != nil || depthAt != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("$path*: %v, %v, %v", got, depthAt, err)
	}
	// C4: one target event, then one closure.hop per attempt: three that
	// resolve and a fourth that finds the chain over.
	if steps != 1+4 {
		t.Errorf("$path* cost %d steps, want 5", steps)
	}

	plus := `{"$path+": ["next"], "from": ["state","list"], "maxDepth": 10}`
	got, _, _, err = closureOf(t, stateList, plus)
	if err != nil || !reflect.DeepEqual(got, want[1:]) {
		t.Errorf("$path+ should be $path* without the base: %v, %v", got, err)
	}

	// C5: a $path+ with no hop is a failure; a $path* is just the base.
	if _, _, _, err = closureOf(t, stateList, `{"$path+": ["next"], "from": ["state","lone"], "maxDepth": 3}`); err == nil || err.Code != "MISSING_PATH" {
		t.Errorf("$path+ with no hop: %v", err)
	}
	got, _, _, err = closureOf(t, stateList, `{"$path*": ["next"], "from": ["state","lone"], "maxDepth": 3}`)
	if err != nil || len(got) != 1 {
		t.Errorf("$path* with no hop: %v, %v", got, err)
	}
	// ...and the base itself must resolve.
	if _, _, _, err = closureOf(t, stateList, `{"$path*": ["next"], "from": ["state","ghost"], "maxDepth": 3}`); err == nil || err.Code != "MISSING_PATH" {
		t.Errorf("missing base: %v", err)
	}
}

func TestClosureHorizon(t *testing.T) {
	p := func(segs ...any) []any { return segs }
	// Three hops exist. maxDepth 3 is exactly enough: no finding.
	_, _, depthAt, err := closureOf(t, stateList, `{"$path*": ["next"], "from": ["state","list"], "maxDepth": 3}`)
	if err != nil || depthAt != nil {
		t.Errorf("maxDepth 3: %v, %v", depthAt, err)
	}
	// maxDepth 2: two hops taken, a third would still resolve (C3).
	got, steps, depthAt, err := closureOf(t, stateList, `{"$path*": ["next"], "from": ["state","list"], "maxDepth": 2}`)
	if err != nil || len(got) != 3 || !reflect.DeepEqual(depthAt, p("state", "list", "next", "next")) {
		t.Fatalf("maxDepth 2: %v, %v, %v", got, depthAt, err)
	}
	if steps != 1+3 { // target, then hops 1 and 2, then the probe past the horizon
		t.Errorf("maxDepth 2 cost %d, want 4", steps)
	}
	// maxDepth 1 on $path+: one node, then the finding.
	got, _, depthAt, _ = closureOf(t, stateList, `{"$path+": ["next"], "from": ["state","list"], "maxDepth": 1}`)
	if len(got) != 1 || depthAt == nil {
		t.Errorf("$path+ maxDepth 1: %v, %v", got, depthAt)
	}
}

func TestClosureFindingReachesTheReport(t *testing.T) {
	rt, _, reg := harness(t, pkgList, stateList, 100000)
	src := func(mode string) string {
		return `{"target": {"$path*": ["next"], "from": ["state","list"], "maxDepth": 2}, "shape": "Node", "mode": "` + mode + `"}`
	}
	rep, err := rt.Run(entry(t, reg, src("report")), false)
	if err != nil || len(rep.Violations) != 1 {
		t.Fatalf("report: %v, %v", rep.Violations, err)
	}
	v := rep.Violations[0]
	if v.Code != CodePathDepthExceeded || v.Message != msgPathDepthExceeded || v.Severity != SeverityViolation || v.Shape != "Node" ||
		!reflect.DeepEqual(v.FocusPath, []any{"state", "list", "next", "next"}) || v.Kind() != "structural" || rep.Conforms() {
		t.Errorf("finding %+v", v)
	}
	// The three nodes inside the horizon were still judged (and conform).
	// Gate raises it, as EXECUTION_ERROR.
	_, err = rt.Run(entry(t, reg, src("gate")), false)
	if err == nil || err.Cat != "EXECUTION_ERROR" || err.Code != CodePathDepthExceeded {
		t.Errorf("gate: %v", err)
	}
}

func TestClosureLocalStep(t *testing.T) {
	// C1: the segment comes from the current node's own data.
	step := `[{"$path": ["local","step","dir"]}]`
	target := `{"$path*": ` + step + `, "from": ["state","walk"], "maxDepth": 10}`
	got, _, _, err := closureOf(t, stateList, target)
	want := [][]any{{"state", "walk"}, {"state", "walk", "b"}, {"state", "walk", "b", "a"}, {"state", "walk", "b", "a", "x"}}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("local.step walk: %v, %v", got, err)
	}
	// A segment that is not a string or non-negative integer is TYPE_ERROR.
	_, _, _, err = closureOf(t, stateList, `{"$path*": `+step+`, "from": ["state","bad"], "maxDepth": 3}`)
	if err == nil || err.Code != "TYPE_ERROR" {
		t.Errorf("array-valued segment: %v", err)
	}
	// Integer segments walk arrays; a multi-segment step is one hop.
	got, steps, _, err := closureOf(t, stateList, `{"$path*": [0], "from": ["state","grid"], "maxDepth": 10}`)
	if err != nil || len(got) != 4 { // grid, grid[0], grid[0][0], grid[0][0][0]
		t.Fatalf("integer step: %v, %v", got, err)
	}
	if steps != 1+4 {
		t.Errorf("cost %d", steps)
	}
	got, _, _, err = closureOf(t, stateList, `{"$path+": ["next","next"], "from": ["state","list"], "maxDepth": 1}`)
	if err != nil || len(got) != 1 || !reflect.DeepEqual(got[0], []any{"state", "list", "next", "next"}) {
		t.Errorf("two-segment step: %v, %v", got, err)
	}
}

func TestClosurePricedByTheCostTable(t *testing.T) {
	rt, m, reg := harness(t, pkgList, stateList, 100000)
	tbl := jaxson.UnitTable()
	tbl.Event[EventClosureHop] = jaxson.Cost{Base: 7}
	if e := m.SetCostTable(tbl); e != nil {
		t.Fatal(e)
	}
	tg := parseTarget("t", val(t, `{"$path*": ["next"], "from": ["state","list"], "maxDepth": 10}`), reg)
	tg.resolve(m, rt.idx)
	if m.Steps() != 1+4*7 {
		t.Errorf("hop pricing: %d steps, want 29", m.Steps())
	}
}

func TestClosureParseRules(t *testing.T) {
	reg := parseReg(t, pkgList)
	mustFail := func(src, cat string) {
		t.Helper()
		if err := raise(func() { parseTarget("t", val(t, src), reg) }); err == nil || err.Cat != cat {
			t.Errorf("%s: %v, want %s", src, err, cat)
		}
	}
	// C6 / section 6: maxDepth is mandatory and a positive integer.
	mustFail(`{"$path*": ["next"], "from": ["state","list"]}`, "PROGRAM_ERROR")
	mustFail(`{"$path+": ["next"], "from": ["state","list"], "maxDepth": 0}`, "PROGRAM_ERROR")
	mustFail(`{"$path+": ["next"], "from": ["state","list"], "maxDepth": -1}`, "PROGRAM_ERROR")
	mustFail(`{"$path+": ["next"], "from": ["state","list"], "maxDepth": 1.5}`, "PROGRAM_ERROR")
	mustFail(`{"$path+": ["next"], "from": ["state","list"], "maxDepth": "3"}`, "PROGRAM_ERROR")
	// Other defects are SHAPE_ERROR.
	mustFail(`{"$path*": ["next"], "maxDepth": 3}`, CatShapeError)                                // a target needs from
	mustFail(`{"$path*": [], "from": ["state","list"], "maxDepth": 3}`, CatShapeError)            // empty step
	mustFail(`{"$path*": "next", "from": ["state","list"], "maxDepth": 3}`, CatShapeError)        // step not an array
	mustFail(`{"$path*": [-1], "from": ["state","list"], "maxDepth": 3}`, CatShapeError)          // negative segment
	mustFail(`{"$path*": [true], "from": ["state","list"], "maxDepth": 3}`, CatShapeError)        // bad segment kind
	mustFail(`{"$path*": ["next"], "from": ["local","x"], "maxDepth": 3}`, CatShapeError)         // bad base root
	mustFail(`{"$path*": ["next"], "from": ["state"], "maxDepth": 3, "extra": 1}`, CatShapeError) // unknown key
	mustFail(`{"$path*": ["next"], "$path+": ["next"], "from": ["state"], "maxDepth": 3}`, CatShapeError)
	mustFail(`{"$path*": [{"$path": ["local","nope","x"]}], "from": ["state"], "maxDepth": 3}`, CatShapeError) // unknown local
	// local.step is in scope inside the step, and nowhere else.
	if err := raise(func() {
		parseTarget("t", val(t, `{"$path*": [{"$path": ["local","step","dir"]}], "from": ["state","walk"], "maxDepth": 3}`), reg)
	}); err != nil {
		t.Errorf("local.step rejected: %v", err)
	}
	mustFail(`{"$path": ["state", {"$path": ["local","step","dir"]}]}`, CatShapeError)
}
