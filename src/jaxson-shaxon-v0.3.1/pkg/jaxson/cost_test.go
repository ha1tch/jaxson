// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the Apache License, Version 2.0.
// https://www.apache.org/licenses/LICENSE-2.0
package jaxson_test

// Tests for the cost table (docs/proposals/step-cost-model.md, CM-1). That
// the `unit` table reproduces the original accounting is pinned separately,
// by TestUnitStepCountsFrozen; these tests cover the table's arithmetic,
// validation, charge-before-effect, how a dialect selects a table, and the
// bound on limits.steps.

import (
	"strings"
	"testing"

	"github.com/ha1tch/jaxson/pkg/jaxson"
)

const maxCost = int64(1<<53 - 1)

func TestCostAtArithmetic(t *testing.T) {
	cases := []struct {
		name string
		c    jaxson.Cost
		size int64
		want int64
	}{
		{"base only", jaxson.Cost{Base: 7}, 1000, 7},
		{"zero size", jaxson.Cost{Base: 2, Per: 3, Div: 1, Of: jaxson.SizeNodes}, 0, 2},
		{"floor division", jaxson.Cost{Base: 1, Per: 1, Div: 2, Of: jaxson.SizeNodes}, 5, 3},
		{"exact division", jaxson.Cost{Base: 1, Per: 1, Div: 2, Of: jaxson.SizeNodes}, 6, 4},
		{"per exceeding div", jaxson.Cost{Per: 7, Div: 3, Of: jaxson.SizeElements}, 10, 23},
		{"zero per ignores div", jaxson.Cost{Base: 4, Per: 0, Div: 0}, 99, 4},
		{"exactly the maximum", jaxson.Cost{Base: maxCost}, 0, maxCost},
		{"base plus size reaches the maximum", jaxson.Cost{Base: maxCost - 5, Per: 1, Div: 1, Of: jaxson.SizeNodes}, 5, maxCost},
		{"just past the maximum saturates", jaxson.Cost{Base: maxCost - 5, Per: 1, Div: 1, Of: jaxson.SizeNodes}, 6, maxCost + 1},
		{"64-bit product overflow saturates", jaxson.Cost{Per: 1 << 40, Div: 1, Of: jaxson.SizeNodes}, 1 << 40, maxCost + 1},
	}
	for _, tc := range cases {
		if got := tc.c.At(tc.size); got != tc.want {
			t.Errorf("%s: At(%d) = %d, want %d", tc.name, tc.size, got, tc.want)
		}
	}
}

// A saturated cost is above every legal limit, so charging it always fails.
func TestSaturatedChargeAlwaysFails(t *testing.T) {
	m := jaxson.NewMachine(nil, map[string]any{}, nil, int(maxCost), nil, nil)
	err := catch(func() { m.Charge(jaxson.Cost{Per: 1 << 40, Div: 1, Of: jaxson.SizeNodes}.At(1 << 40)) })
	if err == nil || err.Cat != "RESOURCE_ERROR" || err.Code != "STEPS" {
		t.Fatalf("saturated charge: %v, want RESOURCE_ERROR/STEPS", err)
	}
	if m.Steps() != 0 {
		t.Fatalf("a failed charge advanced the total to %d", m.Steps())
	}
}

func catch(f func()) (err *jaxson.Err) {
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

func TestCostTableValidation(t *testing.T) {
	ins, ops := jaxson.CoreInstructions(), jaxson.CoreOperators()
	if err := jaxson.UnitTable().Validate(ins, ops); err != nil {
		t.Fatalf("unit table rejected: %v", err)
	}
	one := jaxson.Cost{Base: 1}

	// A table with no fallback must price everything the machine can run.
	bare := jaxson.CostTable{Name: "bare", Instr: map[string]jaxson.Cost{"set": one}, Op: map[string]jaxson.Cost{"add": one}, LoopIter: one}
	err := bare.Validate(ins, ops)
	if err == nil || !strings.Contains(err.Error(), "instruction append") || !strings.Contains(err.Error(), "operator mul") {
		t.Errorf("incomplete table: %v, want the missing instruction and operator named", err)
	}
	full := jaxson.CostTable{Name: "full", Instr: map[string]jaxson.Cost{}, Op: map[string]jaxson.Cost{}, LoopIter: one}
	for n := range ins {
		full.Instr[n] = one
	}
	for n := range ops {
		full.Op[n] = one
	}
	if err := full.Validate(ins, ops); err != nil {
		t.Errorf("complete fallback-less table rejected: %v", err)
	}

	bad := map[string]jaxson.Cost{
		"negative base":     {Base: -1},
		"base past maximum": {Base: maxCost + 1},
		"per without div":   {Per: 1, Of: jaxson.SizeNodes},
		"per without kind":  {Per: 1, Div: 1},
		"div past maximum":  {Per: 1, Div: maxCost + 1, Of: jaxson.SizeNodes},
		"negative per":      {Per: -1, Div: 1, Of: jaxson.SizeNodes},
	}
	for name, c := range bad {
		tbl := jaxson.UnitTable()
		tbl.Event["x"] = c
		if err := tbl.Validate(ins, ops); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

func TestUnitTablesAreIndependent(t *testing.T) {
	a, b := jaxson.UnitTable(), jaxson.UnitTable()
	a.Event["x"] = jaxson.Cost{Base: 9}
	if _, shared := b.Event["x"]; shared {
		t.Fatal("UnitTable returned a shared map")
	}
	*a.Fallback = jaxson.Cost{Base: 9}
	if b.Fallback.Base != 1 {
		t.Fatal("UnitTable returned a shared fallback")
	}
}

func TestResolveCostTable(t *testing.T) {
	absent, err := jaxson.ResolveCostTable(map[string]any{})
	if err != nil || absent.Name != jaxson.UnitName {
		t.Errorf("absent costTable: %v, %v; want unit", absent.Name, err)
	}
	named, err := jaxson.ResolveCostTable(dec(t, `{"limits":{"costTable":"unit"}}`).(map[string]any))
	if err != nil || named.Name != jaxson.UnitName {
		t.Errorf("costTable unit: %v, %v", named.Name, err)
	}
	for _, src := range []string{`{"limits":{"costTable":"nope"}}`, `{"limits":{"costTable":3}}`, `{"limits":{"costTable":null}}`} {
		if _, err := jaxson.ResolveCostTable(dec(t, src).(map[string]any)); err == nil || err.Cat != "VERSION_ERROR" {
			t.Errorf("%s: %v, want VERSION_ERROR", src, err)
		}
	}
}

// weighted is a table deliberately unlike unit, for exercising the machinery.
func weighted() jaxson.CostTable {
	one := jaxson.Cost{Base: 1}
	return jaxson.CostTable{
		Name:     "test-weighted",
		Instr:    map[string]jaxson.Cost{"set": {Base: 4}, "for": {Base: 2}},
		Op:       map[string]jaxson.Cost{"add": {Base: 3}},
		Event:    map[string]jaxson.Cost{"x.y": {Base: 3, Per: 1, Div: 2, Of: jaxson.SizeNodes}},
		LoopIter: jaxson.Cost{Base: 1, Per: 1, Div: 1, Of: jaxson.SizeElements},
		Fallback: &one,
	}
}

func runOn(t *testing.T, tbl jaxson.CostTable, limit int, prog string) (*jaxson.Machine, *jaxson.Err) {
	t.Helper()
	m := jaxson.NewMachine(nil, map[string]any{}, nil, limit, nil, nil)
	if e := m.SetCostTable(tbl); e != nil {
		t.Fatalf("SetCostTable: %v", e)
	}
	return m, catch(func() { m.RunProgram(dec(t, prog).([]any)) })
}

func TestWeightedChargesFollowTheTable(t *testing.T) {
	// set (4) + the add it evaluates (3).
	m, err := runOn(t, weighted(), 1000, `[{"op":"set","path":["state","n"],"value":{"$compute":{"with":{},"expr":["add",1,2]}}}]`)
	if err != nil || m.Steps() != 7 {
		t.Errorf("set+add: steps %d err %v, want 7", m.Steps(), err)
	}
	// for (2) + three iterations of 1 + 3 (the per-element term) each.
	m, err = runOn(t, weighted(), 1000, `[{"op":"for","in":[1,2,3],"as":"x","do":[]}]`)
	if err != nil || m.Steps() != 14 {
		t.Errorf("for over 3: steps %d err %v, want 14", m.Steps(), err)
	}
	// An instruction with no entry costs the fallback.
	m, err = runOn(t, weighted(), 1000, `[{"op":"append","path":["state","a"],"value":1}]`)
	_ = err // append to a missing array is a runtime error; its charge happens first
	if m.Steps() != 1 {
		t.Errorf("fallback instruction: steps %d, want 1", m.Steps())
	}
}

func TestChargeEvent(t *testing.T) {
	m := jaxson.NewMachine(nil, map[string]any{}, nil, 1000, nil, nil)
	if e := m.SetCostTable(weighted()); e != nil {
		t.Fatal(e)
	}
	m.ChargeEvent("x.y", 5) // 3 + floor(5/2)
	if m.Steps() != 5 {
		t.Errorf("sized event: %d, want 5", m.Steps())
	}
	m.ChargeEvent("not.listed", 0) // fallback
	if m.Steps() != 6 {
		t.Errorf("fallback event: %d, want 6", m.Steps())
	}
	if c := m.EventCost("x.y"); c.Per != 1 {
		t.Errorf("EventCost lost the size term: %+v", c)
	}

	// A table with no fallback fails on a dialect event it does not price.
	m2 := jaxson.NewMachine(nil, map[string]any{}, nil, 1000, nil, nil)
	full := jaxson.CostTable{Name: "full", Instr: map[string]jaxson.Cost{}, Op: map[string]jaxson.Cost{}, Event: map[string]jaxson.Cost{}, LoopIter: jaxson.Cost{Base: 1}}
	for n := range jaxson.CoreInstructions() {
		full.Instr[n] = jaxson.Cost{Base: 1}
	}
	for n := range jaxson.CoreOperators() {
		full.Op[n] = jaxson.Cost{Base: 1}
	}
	if e := m2.SetCostTable(full); e != nil {
		t.Fatal(e)
	}
	if err := catch(func() { m2.ChargeEvent("unpriced", 0) }); err == nil || err.Cat != "PROGRAM_ERROR" {
		t.Errorf("unpriced event on a fallback-less table: %v, want PROGRAM_ERROR", err)
	}
	if m2.Steps() != 0 {
		t.Errorf("unpriced event charged %d", m2.Steps())
	}
}

// Charge before effect: the charge that would pass the limit fails, the
// total stays at what was already spent, and the instruction does not run.
func TestChargeBeforeEffect(t *testing.T) {
	m, err := runOn(t, weighted(), 10,
		`[{"op":"set","path":["state","a"],"value":1},{"op":"set","path":["state","b"],"value":2},{"op":"set","path":["state","c"],"value":3}]`)
	if err == nil || err.Cat != "RESOURCE_ERROR" || err.Code != "STEPS" {
		t.Fatalf("err = %v, want RESOURCE_ERROR/STEPS", err)
	}
	if m.Steps() != 8 {
		t.Errorf("total after failure = %d, want 8 (the failing 4 not added)", m.Steps())
	}
	if v := m.GetAt("state", []any{"b"}); v == nil {
		t.Error("second set should have run")
	}
	if _, e := m.Walk("state", []any{"c"}); e == nil {
		t.Error("the set whose charge failed ran anyway")
	}
	// A charge that exactly fills the limit succeeds.
	m, err = runOn(t, weighted(), 12,
		`[{"op":"set","path":["state","a"],"value":1},{"op":"set","path":["state","b"],"value":2},{"op":"set","path":["state","c"],"value":3}]`)
	if err != nil || m.Steps() != 12 {
		t.Errorf("exact fill: steps %d err %v, want 12 and no error", m.Steps(), err)
	}
}

// A non-unit table gives a different total than unit for the same program,
// so the table is demonstrably what is being consulted.
func TestUnitAndWeightedDiffer(t *testing.T) {
	prog := `[{"op":"set","path":["state","n"],"value":{"$compute":{"with":{},"expr":["add",1,2]}}}]`
	u, _ := runOn(t, jaxson.UnitTable(), 1000, prog)
	w, _ := runOn(t, weighted(), 1000, prog)
	if u.Steps() != 2 || w.Steps() != 7 {
		t.Errorf("unit %d (want 2), weighted %d (want 7)", u.Steps(), w.Steps())
	}
}

// costHook is a dialect that owns limits.costTable the way Shaxon will.
type costHook struct {
	jaxson.NoHooks
	p     map[string]any
	m     **jaxson.Machine
	table *jaxson.CostTable // overrides the resolved table when set
}

func (h *costHook) Start(m *jaxson.Machine) {
	t, e := jaxson.ResolveCostTable(h.p)
	if e != nil {
		jaxson.Fail(e.Cat, e.Code, "%s", e.Msg)
	}
	if h.table != nil {
		t = *h.table
	}
	if e := m.SetCostTable(t); e != nil {
		jaxson.Fail(e.Cat, e.Code, "%s", e.Msg)
	}
	*h.m = m
}

func costProfile(m **jaxson.Machine, override *jaxson.CostTable) jaxson.Profile {
	return jaxson.Profile{Key: "toy", Versions: []string{"0.1"}, Limits: []string{"costTable"},
		Begin: func(p map[string]any) jaxson.Hooks { return &costHook{p: p, m: m, table: override} }}
}

func costPkg(t *testing.T, limits string) map[string]any {
	p := corePkg(t)
	delete(p, "jaxson")
	p["toy"] = "0.1"
	p["limits"] = dec(t, limits).(map[string]any)
	return p
}

func TestDialectSelectsTheCostTable(t *testing.T) {
	var m *jaxson.Machine
	// Default: unit.
	if _, err := costProfile(&m, nil).Run(costPkg(t, `{"steps":100}`)); err != nil {
		t.Fatalf("default: %v", err)
	}
	unitSteps := m.Steps()
	// Named unit.
	if _, err := costProfile(&m, nil).Run(costPkg(t, `{"steps":100,"costTable":"unit"}`)); err != nil || m.Steps() != unitSteps {
		t.Errorf("named unit: %d vs %d, err %v", m.Steps(), unitSteps, err)
	}
	// An unknown table is VERSION_ERROR before the program runs.
	m = nil
	if _, err := costProfile(&m, nil).Run(costPkg(t, `{"steps":100,"costTable":"nope"}`)); err == nil || err.Cat != "VERSION_ERROR" {
		t.Errorf("unknown table: %v, want VERSION_ERROR", err)
	}
	// A table the dialect substitutes is the one used.
	w := weighted()
	if _, err := costProfile(&m, &w).Run(costPkg(t, `{"steps":100}`)); err != nil {
		t.Fatalf("weighted: %v", err)
	}
	if m.Steps() == unitSteps {
		t.Errorf("a substituted table left the total unchanged at %d", unitSteps)
	}
	// The core language does not own costTable.
	core := corePkg(t)
	core["limits"] = dec(t, `{"costTable":"unit"}`).(map[string]any)
	if _, err := jaxson.Run(core); err == nil || err.Cat != "VERSION_ERROR" {
		t.Errorf("core accepted costTable: %v", err)
	}
}

func TestStepsLimitBound(t *testing.T) {
	ok := corePkg(t)
	ok["limits"] = dec(t, `{"steps":9007199254740991}`).(map[string]any)
	if _, err := jaxson.Run(ok); err != nil {
		t.Errorf("2^53-1 rejected: %v", err)
	}
	for _, s := range []string{"9007199254740992", "18446744073709551616", "0", "-1", "1.5"} {
		p := corePkg(t)
		p["limits"] = dec(t, `{"steps":`+s+`}`).(map[string]any)
		if _, err := jaxson.Run(p); err == nil || err.Cat != "VERSION_ERROR" {
			t.Errorf("steps %s: %v, want VERSION_ERROR", s, err)
		}
	}
}
