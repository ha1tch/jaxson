// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the Apache License, Version 2.0.
// https://www.apache.org/licenses/LICENSE-2.0
package jaxson

// The cost model of docs/proposals/step-cost-model.md (CM-1).
//
// A run's step total is the sum of the costs of the events it performs, and
// the cost of each event kind comes from a named CostTable, in the way the
// Z80's cost lives in its opcode table. The only table so far is `unit`,
// under which every event costs exactly one step: the original accounting,
// pinned by testdata/unit-step-counts.json.
//
// Rules this file enforces:
//
//   - Charge before effect. Machine.Charge fails with RESOURCE_ERROR/STEPS
//     when the charge would take the total past the limit, and in that case
//     the total is NOT advanced, so "fails at the same point" means the same
//     event on every runtime, even when one charge is larger than one.
//   - Integers, bounded. Costs and limits are non-negative integers no
//     larger than MaxCost (2^53 - 1); a size-dependent term is
//     floor(Per*size/Div) in integer arithmetic and saturates rather than
//     wrapping.
//   - A table is data. UnitTable returns a fresh value on every call, so
//     concurrent runs share nothing.
//   - Nothing here names a dialect. Events a dialect adds are keyed by
//     string in CostTable.Event (Shaxon's are in pkg/shaxon).

import (
	"fmt"
	"math/bits"
	"sort"
	"strings"
)

// MaxCost is the largest legal cost, step limit and step total: 2^53 - 1,
// the largest integer every conformant runtime can represent exactly.
const MaxCost = 1<<53 - 1

// costSaturated is what an overflowing cost computation yields. It is above
// any legal limit, so charging it always fails with RESOURCE_ERROR.
const costSaturated = MaxCost + 1

// SizeKind names the quantity a size-dependent cost term is measured over
// (proposal section 4.1). The Machine does not measure sizes itself: the
// code at a charge site knows the value involved and passes the size in.
type SizeKind int

const (
	// SizeNone: the cost has no size term.
	SizeNone SizeKind = iota
	// SizeNodes: nodes in a value tree that is copied or created.
	SizeNodes
	// SizeElements: items a loop, index build or collection operator visits.
	SizeElements
	// SizeCodepoints: Unicode code points in a string.
	SizeCodepoints
)

// Cost is one table entry: Base + floor(Per*size/Div), the size measured
// over Of. A zero Per means no size term, and then Div and Of are ignored.
type Cost struct {
	Base, Per, Div int64
	Of             SizeKind
}

// At returns the cost for an event of the given size. size is ignored when
// the entry has no size term. A result that would exceed MaxCost saturates
// to a value no legal limit admits.
func (c Cost) At(size int64) int64 {
	total := uint64(c.Base)
	if c.Per != 0 && size > 0 {
		hi, lo := bits.Mul64(uint64(c.Per), uint64(size))
		if hi != 0 {
			return costSaturated
		}
		total += lo / uint64(c.Div)
	}
	if total > MaxCost {
		return costSaturated
	}
	return int64(total)
}

func (c Cost) validate(what string) error {
	switch {
	case c.Base < 0 || c.Base > MaxCost:
		return fmt.Errorf("%s: base %d outside 0..%d", what, c.Base, int64(MaxCost))
	case c.Per < 0 || c.Per > MaxCost:
		return fmt.Errorf("%s: per %d outside 0..%d", what, c.Per, int64(MaxCost))
	case c.Per != 0 && (c.Div < 1 || c.Div > MaxCost):
		return fmt.Errorf("%s: div %d outside 1..%d", what, c.Div, int64(MaxCost))
	case c.Per != 0 && c.Of == SizeNone:
		return fmt.Errorf("%s: a size term needs a size kind", what)
	}
	return nil
}

// CostTable maps each kind of countable event to its Cost (proposal section
// 4.3). Instr and Op are keyed by the instruction / operator name, Event by
// a dialect's own event name (for example "shape.activation"). An event with
// no entry costs Fallback when one is set; a table with no Fallback must list
// every instruction and operator the machine can run (SetCostTable checks).
type CostTable struct {
	Name     string
	Instr    map[string]Cost
	Op       map[string]Cost
	Event    map[string]Cost
	LoopIter Cost
	Fallback *Cost
}

// UnitName is the name of the default table.
const UnitName = "unit"

// UnitTable returns the `unit` table: every event costs one step. It is the
// language's original accounting and the default when a package names no
// table. A fresh value is built on every call.
func UnitTable() CostTable {
	one := Cost{Base: 1}
	return CostTable{
		Name:     UnitName,
		Instr:    map[string]Cost{},
		Op:       map[string]Cost{},
		Event:    map[string]Cost{},
		LoopIter: one,
		Fallback: &one,
	}
}

// LookupCostTable returns the named table, or false if no such table is
// defined by this runtime.
func LookupCostTable(name string) (CostTable, bool) {
	if name == UnitName {
		return UnitTable(), true
	}
	return CostTable{}, false
}

// ResolveCostTable reads limits.costTable from a package. An absent member
// means `unit`; a non-string or unknown name is VERSION_ERROR (an
// unsupported limit), to be raised before the program starts. It is for a
// dialect to call: the core language does not own the member, so a dialect
// lists "costTable" in Profile.Limits and applies the result in its Start
// hook with Machine.SetCostTable.
func ResolveCostTable(p map[string]any) (t CostTable, err *Err) {
	limits, _ := p["limits"].(map[string]any)
	v, has := limits["costTable"]
	if !has {
		return UnitTable(), nil
	}
	name, ok := v.(string)
	if !ok {
		return CostTable{}, &Err{Cat: "VERSION_ERROR", Msg: "costTable must be a string"}
	}
	t, found := LookupCostTable(name)
	if !found {
		return CostTable{}, &Err{Cat: "VERSION_ERROR", Msg: fmt.Sprintf("unsupported costTable %q", name)}
	}
	return t, nil
}

// Validate checks every entry's arithmetic bounds and, when the table has
// no Fallback, that it covers all the given instruction and operator names.
func (t CostTable) Validate(instructions map[string]InstructionDef, operators map[string]OperatorDef) error {
	check := func(kind string, m map[string]Cost) error {
		for _, k := range sortedCostKeys(m) {
			if err := m[k].validate(kind + " " + k); err != nil {
				return err
			}
		}
		return nil
	}
	for _, e := range []error{check("instruction", t.Instr), check("operator", t.Op), check("event", t.Event)} {
		if e != nil {
			return e
		}
	}
	if err := t.LoopIter.validate("loop iteration"); err != nil {
		return err
	}
	if t.Fallback != nil {
		return t.Fallback.validate("fallback")
	}
	var missing []string
	for name := range instructions {
		if _, ok := t.Instr[name]; !ok {
			missing = append(missing, "instruction "+name)
		}
	}
	for name := range operators {
		if _, ok := t.Op[name]; !ok {
			missing = append(missing, "operator "+name)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return fmt.Errorf("cost table %q has no entry for: %s", t.Name, strings.Join(missing, ", "))
	}
	return nil
}

func sortedCostKeys(m map[string]Cost) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// lookup returns the entry for name in m, or the fallback. A miss with no
// fallback cannot happen after SetCostTable's validation for a registered
// instruction or operator; for a dialect event it is a programming error
// in the dialect, raised as such rather than charged as 1.
func (t CostTable) lookup(kind, name string, m map[string]Cost) Cost {
	if c, ok := m[name]; ok {
		return c
	}
	if t.Fallback != nil {
		return *t.Fallback
	}
	fail("PROGRAM_ERROR", "", "cost table %q has no entry for %s %q", t.Name, kind, name)
	return Cost{}
}

// ---------------------------------------------------------------- Machine

// Steps returns the step total charged so far.
func (m *Machine) Steps() int64 { return m.steps }

// SetCostTable installs a cost table for this machine, validating it
// against the machine's own instruction and operator tables. Call it before
// anything runs (a dialect's Start hook). The default is UnitTable.
func (m *Machine) SetCostTable(t CostTable) *Err {
	if err := t.Validate(m.instructions, m.operators); err != nil {
		return &Err{Cat: "VERSION_ERROR", Msg: err.Error()}
	}
	m.costs = t
	m.cache, m.ccache = nil, nil // compiled code embeds the costs
	return nil
}

// Charge adds n steps, failing with RESOURCE_ERROR/STEPS if the total would
// pass the limit. On failure the total is not advanced: charge before
// effect (proposal principle 3).
func (m *Machine) Charge(n int64) {
	if n < 0 {
		fail("PROGRAM_ERROR", "", "negative step charge %d", n)
	}
	if n > m.limit-m.steps {
		fail("RESOURCE_ERROR", "STEPS", "step limit %d exceeded", m.limit)
	}
	m.steps += n
}

// EventCost returns the table entry for a dialect event, so a charge site
// can skip measuring a size the entry does not use (Per == 0).
func (m *Machine) EventCost(name string) Cost {
	return m.costs.lookup("event", name, m.costs.Event)
}

// ChargeEvent charges the cost of a dialect event of the given size.
func (m *Machine) ChargeEvent(name string, size int64) {
	m.Charge(m.EventCost(name).At(size))
}
