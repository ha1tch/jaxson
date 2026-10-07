// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the GNU General Public License, version 3.
// https://www.gnu.org/licenses/gpl-3.0.html
package jaxson

import "reflect"

// ConcatKey compiles x, when it is a `$compute` island whose expression is a
// single `concat` of string literals and `{"$v": name}` references, to a
// function that appends the result to dst instead of returning a new string.
// A host that builds many keys of this shape (an index over a long trail)
// writes them into one byte slab and allocates nothing per key.
//
// The function does exactly what evaluating the island would: the `with`
// bindings are read in the same order and charge the same steps, the
// operator is charged once, and a part that is not a string fails with the
// same TYPE_ERROR. It returns false when x is any other shape, or when the
// host has replaced `concat` or the `$compute` form, so the caller keeps the
// ordinary path. The function may be called only while no other island of
// this machine is part-way through evaluation, which holds between the
// elements of a host's loop; it leaves the scratch stack as it found it
// unless it fails, in which case the host's recovery restores it as it does
// for Eval.
func (m *Machine) ConcatKey(x any) (func(dst []byte) []byte, bool) {
	if m.tree {
		return nil, false
	}
	mp, ok := x.(map[string]any)
	if !ok || len(mp) != 1 {
		return nil, false
	}
	if _, over := formOf(m.forms, mp); over {
		return nil, false
	}
	cm, ok := mp["$compute"].(map[string]any)
	if !ok {
		return nil, false
	}
	ex, ok := cm["expr"].([]any)
	if !ok || len(ex) < 2 || ex[0] != "concat" {
		return nil, false
	}
	d, known := m.operators["concat"]
	core := CoreOperators()["concat"]
	if !known || d.Fn == nil || reflect.ValueOf(d.Fn).Pointer() != reflect.ValueOf(core.Fn).Pointer() {
		return nil, false
	}
	c := &Compiler{m}
	slots := map[string]int{}
	var withs []func(*Machine) any
	if w, ok := cm["with"].(map[string]any); ok {
		for i, n := range SortedKeys(w) {
			slots[n] = i
			withs = append(withs, c.Operand(w[n]))
		}
	}
	type part struct {
		lit  string
		slot int // -1: a literal; -2: an unbound name, which reads as null
	}
	parts := make([]part, 0, len(ex)-1)
	for _, a := range ex[1:] {
		switch t := a.(type) {
		case string:
			parts = append(parts, part{lit: t, slot: -1})
		case map[string]any:
			name, isV := t["$v"].(string)
			if !isV || len(t) != 1 {
				return nil, false
			}
			if i, ok := slots[name]; ok {
				parts = append(parts, part{slot: i})
			} else {
				parts = append(parts, part{slot: -2})
			}
		default:
			return nil, false
		}
	}
	cost, lazy := m.opCost("concat")
	n := len(withs)
	return func(dst []byte) []byte {
		base := m.sp
		env := m.alloc(n)
		for i, w := range withs {
			env[i] = w(m)
		}
		if lazy {
			m.Charge(m.costs.lookup("operator", "concat", m.costs.Op).At(0))
		} else {
			m.Charge(cost)
		}
		// The first part that is not a string fails, as in the operator.
		for _, p := range parts {
			switch p.slot {
			case -1:
				dst = append(dst, p.lit...)
			case -2:
				str(nil)
			default:
				dst = append(dst, str(env[p.slot])...)
			}
		}
		m.release(base)
		return dst
	}, true
}
