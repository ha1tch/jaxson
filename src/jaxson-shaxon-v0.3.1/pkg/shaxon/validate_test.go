// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the GNU General Public License, version 3.
// https://www.gnu.org/licenses/gpl-3.0.html
package shaxon

// Phase 4.2 tests: targets, the validate list, unique, and the check
// instruction (core sections 7 and 8; decisions T1..T5 and V1..V7).

import (
	"reflect"
	"strings"
	"testing"

	"github.com/ha1tch/jaxson/pkg/jaxson"
)

const pkgV = `{
  "indices": {
    "tagsById": {"source": {"$path": ["state", "tags"]}, "key": {"$path": ["local", "item", "id"]}},
    "itemsById": {"source": {"$path": ["state", "items"]}, "key": {"$path": ["local", "item", "id"]}}
  },
  "shapes": {
    "Item": {"kind": "object", "required": ["id"], "closed": false, "fields": {
      "id":  {"kind": "number"},
      "tag": {"kind": "reference", "index": "tagsById"}
    }},
    "Plain": {"kind": "object", "closed": false, "fields": {"id": {"kind": "number"}}},
    "NeedsKind": {"kind": "object", "required": ["kind"], "closed": false}
  }
}`

const stateV = `{
  "tags": [{"id": "a"}, {"id": "b"}],
  "log": [],
  "items": [
    {"id": 1, "tag": "a", "kind": "x", "sku": "s1"},
    {"id": 2, "tag": "zz", "kind": "y", "sku": "s2"},
    {"id": 3, "tag": "b", "kind": "x", "sku": "s1"}
  ],
  "byName": {"m": {"id": 7}, "c": {"id": 8}, "k": "scalar"}
}`

// harness builds a bound Runtime and Machine the way a Shaxon run will.
func harness(t *testing.T, pkg, state string, limit int) (*Runtime, *jaxson.Machine, *Registries) {
	t.Helper()
	reg := parseReg(t, pkg)
	rt := NewRuntime(reg, 0)
	m := jaxson.NewMachine(nil, state0(t, state), map[string]any{}, limit, rt.Instructions(), nil)
	rt.Bind(m)
	return rt, m, reg
}

func entry(t *testing.T, reg *Registries, src string) Entry {
	t.Helper()
	var e Entry
	if err := raise(func() { e = parseEntry("test", val(t, src).(map[string]any), reg) }); err != nil {
		t.Fatalf("entry %s: %v", src, err)
	}
	return e
}

func entryErr(t *testing.T, reg *Registries, src string) *jaxson.Err {
	t.Helper()
	return raise(func() { parseEntry("test", val(t, src).(map[string]any), reg) })
}

func paths(foci []Focus) [][]any {
	out := make([][]any, len(foci))
	for i, f := range foci {
		out[i] = f.Path
	}
	return out
}

func TestTargetForms(t *testing.T) {
	rt, m, reg := harness(t, pkgV, stateV, 100000)
	resolve := func(target string) ([][]any, int64, *jaxson.Err) {
		tg := Target{}
		if err := raise(func() { tg = parseTarget("t", val(t, target), reg) }); err != nil {
			t.Fatalf("parse %s: %v", target, err)
		}
		before := m.Steps()
		var foci []Focus
		err := raise(func() { foci, _ = tg.resolve(m, rt.idx) })
		return paths(foci), m.Steps() - before, err
	}
	p := func(segs ...any) []any { return segs }

	got, steps, err := resolve(`{"$path": ["state", "items", 1]}`)
	if err != nil || !reflect.DeepEqual(got, [][]any{p("state", "items", 1)}) || steps != 1 {
		t.Errorf("$path: %v, %d steps, %v", got, steps, err)
	}

	got, steps, err = resolve(`{"$each": ["state", "items"]}`)
	if err != nil || len(got) != 3 || got[2][2] != 2 || steps != 1 {
		t.Errorf("$each array: %v, %d steps, %v", got, steps, err)
	}

	// T4: object members in code-point key order.
	got, _, err = resolve(`{"$each": ["state", "byName"]}`)
	want := [][]any{p("state", "byName", "c"), p("state", "byName", "k"), p("state", "byName", "m")}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("$each object: %v, %v", got, err)
	}

	// T3: the population is the elements whose field equals the value.
	got, steps, err = resolve(`{"$discriminator": {"at": ["state", "items"], "field": "kind", "value": "x"}}`)
	want = [][]any{p("state", "items", 0), p("state", "items", 2)}
	if err != nil || !reflect.DeepEqual(got, want) || steps != 1 {
		t.Errorf("$discriminator: %v, %d steps, %v", got, steps, err)
	}
	// Non-objects and elements lacking the field are not in it.
	got, _, err = resolve(`{"$discriminator": {"at": ["state", "byName"], "field": "id", "value": 7}}`)
	if err != nil || !reflect.DeepEqual(got, [][]any{p("state", "byName", "m")}) {
		t.Errorf("$discriminator over mixed members: %v, %v", got, err)
	}

	got, steps, err = resolve(`{"$indexed": "itemsById"}`)
	want = [][]any{p("state", "items", 0), p("state", "items", 1), p("state", "items", 2)}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("$indexed: %v, %v", got, err)
	}
	if steps != 3+1 { // the 3-element index build, then the target's own event
		t.Errorf("$indexed cost %d, want 4", steps)
	}

	// T2, and a missing path.
	if _, _, err = resolve(`{"$each": ["state", "tags", 0, "id"]}`); err == nil || err.Code != "TYPE_ERROR" {
		t.Errorf("$each over a scalar: %v", err)
	}
	if _, _, err = resolve(`{"$path": ["state", "nope"]}`); err == nil || err.Code != "MISSING_PATH" {
		t.Errorf("$path missing: %v", err)
	}
}

func TestTargetPricedByTheCostTable(t *testing.T) {
	rt, m, reg := harness(t, pkgV, stateV, 100000)
	tbl := jaxson.UnitTable()
	tbl.Event[EventTargetResolve] = jaxson.Cost{Base: 10, Per: 2, Div: 1, Of: jaxson.SizeElements}
	if e := m.SetCostTable(tbl); e != nil {
		t.Fatal(e)
	}
	tg := parseTarget("t", val(t, `{"$each": ["state", "items"]}`), reg)
	tg.resolve(m, rt.idx)
	if m.Steps() != 10+2*3 {
		t.Errorf("sized target cost %d, want 16", m.Steps())
	}
}

func TestTargetParseErrors(t *testing.T) {
	reg := parseReg(t, pkgV)
	for _, src := range []string{
		`{"$path": ["state"], "$each": ["state"]}`, // two forms
		`{"$nope": ["state"]}`,
		`{"$path": []}`,
		`{"$path": ["local", "x"]}`, // not a root a target may start at
		`{"$each": "state"}`,
		`{"$indexed": "ghost"}`,
		`{"$indexed": 3}`,
		`{"$discriminator": {"at": ["state","items"], "field": "kind"}}`,
		`{"$discriminator": {"at": ["state","items"], "value": 1}}`,
		`{"$discriminator": {"field": "kind", "value": 1}}`,
		`{"$discriminator": {"at": ["state","items"], "field": "kind", "value": 1, "extra": 2}}`,
		`"state"`,
	} {
		if err := raise(func() { parseTarget("t", val(t, src), reg) }); err == nil || err.Cat != CatShapeError {
			t.Errorf("%s: %v, want a SHAPE_ERROR", src, err)
		}
	}
}

func TestEntryParseRules(t *testing.T) {
	reg := parseReg(t, pkgV)
	good := `{"target": {"$path": ["state"]}, "shape": "Plain", "mode": "gate"}`
	if e := entry(t, reg, good); e.Mode != ModeGate || e.Shape != "Plain" {
		t.Errorf("parsed %+v", e)
	}
	bad := map[string]string{
		"both shape and unique":      `{"target": {"$path": ["state"]}, "shape": "Plain", "unique": {}, "mode": "gate"}`,
		"neither":                    `{"target": {"$path": ["state"]}, "mode": "gate"}`,
		"no mode (V1)":               `{"target": {"$path": ["state"]}, "shape": "Plain"}`,
		"bad mode":                   `{"target": {"$path": ["state"]}, "shape": "Plain", "mode": "loud"}`,
		"into with gate (V2)":        `{"target": {"$path": ["state"]}, "shape": "Plain", "mode": "gate", "into": ["state", "log"]}`,
		"into at input":              `{"target": {"$path": ["state"]}, "shape": "Plain", "mode": "report", "into": ["input", "x"]}`,
		"empty into":                 `{"target": {"$path": ["state"]}, "shape": "Plain", "mode": "report", "into": []}`,
		"unknown key":                `{"target": {"$path": ["state"]}, "shape": "Plain", "mode": "gate", "extra": 1}`,
		"undeclared shape":           `{"target": {"$path": ["state"]}, "shape": "Ghost", "mode": "gate"}`,
		"severity on a shape (V6)":   `{"target": {"$path": ["state"]}, "shape": "Plain", "mode": "gate", "severity": "warning"}`,
		"message on a shape (V6)":    `{"target": {"$path": ["state"]}, "shape": "Plain", "mode": "gate", "message": "x"}`,
		"unique with a bad member":   `{"target": {"$path": ["state"]}, "unique": {"fields": "a"}, "mode": "gate"}`,
		"unique with a bad severity": `{"target": {"$path": ["state"]}, "unique": {}, "severity": "loud", "mode": "gate"}`,
		"no target":                  `{"shape": "Plain", "mode": "gate"}`,
	}
	for name, src := range bad {
		if err := entryErr(t, reg, src); err == nil || err.Cat != CatShapeError {
			t.Errorf("%s: %v, want a SHAPE_ERROR", name, err)
		}
	}
	u := entry(t, reg, `{"target": {"$each": ["state","items"]}, "unique": {"field": "sku", "id": "SKU"}, "severity": "warning", "message": "dup", "mode": "report"}`)
	if u.Unique == nil || u.Unique.Field != "sku" || u.Unique.ID != "SKU" || u.Unique.Severity != "warning" || u.Unique.Message != "dup" {
		t.Errorf("unique parsed as %+v", u.Unique)
	}
}

func TestParseValidateList(t *testing.T) {
	reg := parseReg(t, pkgV)
	pkg := map[string]any{"validate": val(t, `[
	  {"target": {"$path": ["state"]}, "shape": "Plain", "mode": "gate"},
	  {"target": {"$each": ["state", "items"]}, "unique": {"field": "sku"}, "mode": "report"}]`)}
	es, err := ParseValidate(pkg, reg)
	if err != nil || len(es) != 2 || es[1].Unique == nil {
		t.Fatalf("ParseValidate: %v, %v", es, err)
	}
	if es, err := ParseValidate(map[string]any{}, reg); err != nil || es != nil {
		t.Errorf("absent validate: %v, %v", es, err)
	}
	for _, src := range []string{`{"validate": {}}`, `{"validate": [3]}`, `{"validate": [{"target": {"$path": ["state"]}, "mode": "gate"}]}`} {
		if _, err := ParseValidate(val(t, src).(map[string]any), reg); err == nil || err.Cat != CatShapeError || !strings.Contains(err.Msg, "validate") {
			t.Errorf("%s: %v", src, err)
		}
	}
}

func TestUniqueSemantics(t *testing.T) {
	rt, _, reg := harness(t, pkgV, stateV, 100000)
	run := func(src string) (Report, *jaxson.Err) { return rt.Run(entry(t, reg, src), false) }

	// s1 repeats (items 0 and 2): only the later one is flagged (core section 7).
	rep, err := run(`{"target": {"$each": ["state","items"]}, "unique": {"field": "sku", "id": "SKU"}, "mode": "report"}`)
	if err != nil || len(rep.Violations) != 1 {
		t.Fatalf("unique: %v, %v", rep.Violations, err)
	}
	v := rep.Violations[0]
	if !reflect.DeepEqual(v.FocusPath, []any{"state", "items", 2}) || v.ConstraintPath != nil || v.ConstraintID != "SKU" || v.Shape != "" || v.Severity != SeverityViolation {
		t.Errorf("violation %+v", v)
	}
	if val := v.Value(); val["shape"] != nil || val["constraintId"] != "SKU" || val["kind"] != "constraint" {
		t.Errorf("rendered %v", val)
	}
	if _, has := v.Value()["constraintPath"]; has {
		t.Error("constraintPath must be absent")
	}

	// The element itself is the key when no field is named (and 1 == 1.0).
	rt2, _, reg2 := harness(t, pkgV, `{"xs": [1, 2, 1.0, "1", 2], "tags": [], "items": []}`, 1000)
	rep, err = rt2.Run(entry(t, reg2, `{"target": {"$each": ["state","xs"]}, "unique": {}, "mode": "report"}`), false)
	if err != nil || len(rep.Violations) != 2 || rep.Violations[0].FocusPath[2] != 2 || rep.Violations[1].FocusPath[2] != 4 {
		t.Errorf("element-as-key: %v, %v", rep.Violations, err)
	}

	// V4: an object lacking the field has no key and is skipped.
	rt3, _, reg3 := harness(t, pkgV, `{"xs": [{"a": 1}, {"b": 1}, {"a": 1}, {"b": 2}], "tags": [], "items": []}`, 1000)
	rep, _ = rt3.Run(entry(t, reg3, `{"target": {"$each": ["state","xs"]}, "unique": {"field": "a"}, "mode": "report"}`), false)
	if len(rep.Violations) != 1 || rep.Violations[0].FocusPath[2] != 2 {
		t.Errorf("missing field: %v", rep.Violations)
	}

	// Severity and message come from the entry; a warning never gates.
	rep, err = run(`{"target": {"$each": ["state","items"]}, "unique": {"field": "sku"}, "severity": "warning", "message": "duplicate sku", "mode": "gate"}`)
	if err != nil || len(rep.Violations) != 1 || rep.Violations[0].Severity != SeverityWarning || rep.Violations[0].Message != "duplicate sku" || !rep.Conforms() {
		t.Errorf("warning in gate mode: %v, %v", rep.Violations, err)
	}

	// Gate: the first repeat in visiting order aborts.
	_, err = run(`{"target": {"$each": ["state","items"]}, "unique": {"field": "sku"}, "mode": "gate"}`)
	if err == nil || err.Cat != CatValidationError || err.Code != CodeShapeMismatch || !strings.Contains(err.Msg, "items[2]") {
		t.Errorf("gate: %v", err)
	}
}

func TestShapeEntryAcrossFoci(t *testing.T) {
	rt, _, reg := harness(t, pkgV, stateV, 100000)
	// Report mode collects across every focus node: items[1] dangles on "zz".
	rep, err := rt.Run(entry(t, reg, `{"target": {"$each": ["state","items"]}, "shape": "Item", "mode": "report"}`), false)
	if err != nil || len(rep.Violations) != 1 || rep.Violations[0].Code != CodeDanglingReference || rep.Violations[0].FocusPath[2] != 1 {
		t.Fatalf("report: %v, %v", rep.Violations, err)
	}
	// Gate mode stops at the first failing focus and delivers nothing.
	_, err = rt.Run(entry(t, reg, `{"target": {"$each": ["state","items"]}, "shape": "Item", "mode": "gate"}`), false)
	if err == nil || err.Code != CodeDanglingReference {
		t.Errorf("gate: %v", err)
	}
	// A conforming population passes.
	rep, err = rt.Run(entry(t, reg, `{"target": {"$each": ["state","items"]}, "shape": "Plain", "mode": "gate"}`), false)
	if err != nil || len(rep.Violations) != 0 {
		t.Errorf("Plain: %v, %v", rep.Violations, err)
	}
}

func TestDeliveryCombinedSetAndAppend(t *testing.T) {
	rt, m, reg := harness(t, pkgV, stateV, 100000)
	noInto := `{"target": {"$each": ["state","items"]}, "shape": "Item", "mode": "report"}`
	uniq := `{"target": {"$each": ["state","items"]}, "unique": {"field": "sku"}, "mode": "report"}`
	rt.Run(entry(t, reg, noInto), false)
	rt.Run(entry(t, reg, uniq), false)
	// Entries with no into concatenate in execution order (section 7).
	c := rt.Combined()
	if len(c.Violations) != 2 || c.Violations[0].Code != CodeDanglingReference || c.Violations[1].Code != "" {
		t.Errorf("combined: %v", c.Violations)
	}
	if c.Conforms() {
		t.Error("combined report should not conform")
	}

	// V3: validate sets the whole report value at `into`...
	rt.Run(entry(t, reg, `{"target": {"$each": ["state","items"]}, "shape": "Item", "mode": "report", "into": ["state","rep"]}`), false)
	got := jaxson.Legacy(m.GetAt("state", []any{"rep"})).(map[string]any)
	if got["conforms"] != false || len(got["violations"].([]any)) != 1 {
		t.Errorf("set delivery: %v", got)
	}
	// ...and check appends each violation to the array there.
	app := entry(t, reg, `{"target": {"$each": ["state","items"]}, "shape": "Item", "mode": "report", "into": ["state","log"]}`)
	before := m.Steps()
	rt.Run(app, true)
	rt.Run(app, true)
	log := m.GetAt("state", []any{"log"}).([]any)
	if len(log) != 2 {
		t.Fatalf("append delivery: %d entries, want 2", len(log))
	}
	if jaxson.Legacy(log[0]).(map[string]any)["code"] != CodeDanglingReference {
		t.Errorf("appended %v", log[0])
	}
	if m.Steps()-before == 0 {
		t.Error("delivery should cost steps")
	}
}

// V7: the index an entry uses is built before its target resolves, so a
// target that fails still leaves the build charged; and two entries over the
// same index build it once.
func TestIndexRefreshTimingAndSharing(t *testing.T) {
	rt, m, reg := harness(t, pkgV, stateV, 100000)
	_, err := rt.Run(entry(t, reg, `{"target": {"$path": ["state","nope"]}, "shape": "Item", "mode": "gate"}`), false)
	if err == nil || err.Code != "MISSING_PATH" {
		t.Fatalf("err %v", err)
	}
	if m.Steps() != 2 { // tagsById has 2 elements; the missing target costs nothing
		t.Errorf("steps %d, want 2: the index should be built before the target resolves", m.Steps())
	}

	rt, m, reg = harness(t, pkgV, stateV, 100000)
	e := entry(t, reg, `{"target": {"$path": ["state","items",0]}, "shape": "Item", "mode": "gate"}`)
	rt.Run(e, false)
	first := m.Steps()
	rt.Run(e, false)
	second := m.Steps() - first
	if second >= first {
		t.Errorf("second run cost %d, first %d: the index build should have been reused", second, first)
	}
}

// ---------------------------------------------------------------- the instruction

func runProgram(t *testing.T, pkg, state, prog string, limit int) (*Runtime, *jaxson.Machine, *jaxson.Err) {
	t.Helper()
	rt, m, reg := harness(t, pkg, state, limit)
	instr := rt.Instructions()
	p := val(t, prog).([]any)
	if err := jaxson.CheckProgramHost(p, instr, reg); err != nil {
		return rt, m, err
	}
	return rt, m, raise(func() { m.RunProgram(p) })
}

func TestCheckInstruction(t *testing.T) {
	// Gate: a failing check aborts the program; later instructions do not run.
	_, m, err := runProgram(t, pkgV, stateV, `[
	  {"op":"set","path":["state","before"],"value":1},
	  {"op":"check","target":{"$each":["state","items"]},"shape":"Item","mode":"gate"},
	  {"op":"set","path":["state","after"],"value":1}]`, 100000)
	if err == nil || err.Code != CodeDanglingReference {
		t.Fatalf("gate check: %v", err)
	}
	if _, e := m.Walk("state", []any{"before"}); e != nil {
		t.Error("the instruction before the check should have run")
	}
	if _, e := m.Walk("state", []any{"after"}); e == nil {
		t.Error("the instruction after a failed gate check ran")
	}

	// Report with into appends and the program continues.
	_, m, err = runProgram(t, pkgV, stateV, `[
	  {"op":"check","target":{"$each":["state","items"]},"shape":"Item","mode":"report","into":["state","log"]},
	  {"op":"set","path":["state","after"],"value":1}]`, 100000)
	if err != nil {
		t.Fatal(err)
	}
	if n := len(m.GetAt("state", []any{"log"}).([]any)); n != 1 {
		t.Errorf("log has %d entries, want 1", n)
	}
	if _, e := m.Walk("state", []any{"after"}); e != nil {
		t.Error("program stopped after a report-mode check")
	}

	// A target path may use a loop's local, and the check is repeated per turn.
	_, m, err = runProgram(t, pkgV, stateV, `[
	  {"op":"for","in":[0,1,2],"as":"i","do":[
	    {"op":"check","target":{"$path":["state","items",{"$path":["local","i"]}]},"shape":"Plain","mode":"gate"}]}]`, 100000)
	if err != nil {
		t.Errorf("check in a loop: %v", err)
	}

	// The check sees the current state: a mutation between two checks is seen.
	_, _, err = runProgram(t, pkgV, stateV, `[
	  {"op":"check","target":{"$path":["state","items",1]},"shape":"Plain","mode":"gate"},
	  {"op":"set","path":["state","items",1,"id"],"value":"not a number"},
	  {"op":"check","target":{"$path":["state","items",1]},"shape":"Plain","mode":"gate"}]`, 100000)
	if err == nil || err.Cat != CatValidationError {
		t.Errorf("second check should fail on the mutated state: %v", err)
	}
}

func TestCheckInstructionStaticRules(t *testing.T) {
	reg := parseReg(t, pkgV)
	rt := NewRuntime(reg, 0)
	bad := map[string]string{
		"undeclared shape": `[{"op":"check","target":{"$path":["state"]},"shape":"Ghost","mode":"gate"}]`,
		"missing mode":     `[{"op":"check","target":{"$path":["state"]},"shape":"Plain"}]`,
		"both":             `[{"op":"check","target":{"$path":["state"]},"shape":"Plain","unique":{},"mode":"gate"}]`,
		"unknown local":    `[{"op":"check","target":{"$path":["state","items",{"$path":["local","nope"]}]},"shape":"Plain","mode":"gate"}]`,
		"unknown field":    `[{"op":"check","target":{"$path":["state"]},"shape":"Plain","mode":"gate","bogus":1}]`,
	}
	for name, src := range bad {
		if err := jaxson.CheckProgramHost(val(t, src).([]any), rt.Instructions(), reg); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	// The core language does not have it.
	err := jaxson.CheckProgram(val(t, `[{"op":"check","target":{"$path":["state"]},"shape":"Plain","mode":"gate"}]`).([]any), jaxson.CoreInstructions())
	if err == nil {
		t.Error("core accepted a check instruction")
	}
	// And a Shaxon table without a Shaxon host refuses it with a clear error.
	err = jaxson.CheckProgram(val(t, `[{"op":"check","target":{"$path":["state"]},"shape":"Plain","mode":"gate"}]`).([]any), rt.Instructions())
	if err == nil || !strings.Contains(err.Msg, "host") {
		t.Errorf("no host: %v", err)
	}
}

// Validation is read-only apart from a declared `into`: a gate entry that
// passes leaves every root as it found it.
func TestValidationDoesNotWrite(t *testing.T) {
	rt, m, reg := harness(t, pkgV, stateV, 100000)
	before := jaxson.Clone(m.GetAt("state", nil))
	if _, err := rt.Run(entry(t, reg, `{"target": {"$each": ["state","items"]}, "shape": "Plain", "mode": "gate"}`), false); err != nil {
		t.Fatal(err)
	}
	rt.Run(entry(t, reg, `{"target": {"$each": ["state","items"]}, "unique": {"field": "sku"}, "mode": "report"}`), false)
	if !jaxson.Equal(before, m.GetAt("state", nil)) {
		t.Error("a validation entry changed state")
	}
}
