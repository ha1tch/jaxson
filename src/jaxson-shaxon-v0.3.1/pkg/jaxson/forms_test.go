// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the GNU General Public License, version 3.
// https://www.gnu.org/licenses/gpl-3.0.html
package jaxson_test

// Direct tests of the operand-form hook (forms.go). Shaxon exercises it
// through its four real forms; these use a toy form so the hook's own
// contract is pinned independently of any dialect.

import (
	"math/big"
	"testing"

	"github.com/ha1tch/jaxson/pkg/jaxson"
)

// toyForms registers `$twice`: {"$twice": <operand>, "plus": <int literal>}.
// "plus" is an extra key, which a core form may not carry but a registered
// form may. checked records each form object the static check was handed.
func toyForms(checked *[]map[string]any) map[string]jaxson.FormDef {
	return map[string]jaxson.FormDef{
		"$twice": {
			Check: func(f map[string]any, c *jaxson.Checker) {
				if checked != nil {
					*checked = append(*checked, f)
				}
				for k := range f {
					if k != "$twice" && k != "plus" {
						jaxson.Fail("PROGRAM_ERROR", "", "$twice: unknown key %q", k)
					}
				}
				var locals []string
				for n := range c.Locals {
					locals = append(locals, n)
				}
				if e := jaxson.CheckOperandWith(toyForms(nil), f["$twice"], locals...); e != nil {
					jaxson.Fail("PROGRAM_ERROR", "", "$twice: %s", e.Msg)
				}
			},
			Eval: func(m *jaxson.Machine, f map[string]any) any {
				v, ok := m.Eval(f["$twice"]).(*big.Rat)
				if !ok {
					jaxson.Fail("EXECUTION_ERROR", "TYPE_ERROR", "$twice needs a number")
				}
				out := new(big.Rat).Mul(v, big.NewRat(2, 1))
				if p, has := f["plus"].(*big.Rat); has {
					out.Add(out, p)
				}
				return out
			},
		},
	}
}

func TestCoreRejectsAFormItDoesNotKnow(t *testing.T) {
	err := jaxson.CheckProgram(dec(t, `[{"op":"set","path":["state","n"],"value":{"$twice":2}}]`), jaxson.CoreInstructions())
	if err == nil || err.Cat != "PROGRAM_ERROR" {
		t.Fatalf("core accepted an unregistered form: %v", err)
	}
	if e := jaxson.CheckOperand(dec(t, `{"$twice":2}`)); e == nil {
		t.Fatalf("CheckOperand accepted an unregistered form")
	}
}

func TestRegisteredFormIsCheckedWithItsWholeObject(t *testing.T) {
	var checked []map[string]any
	forms := toyForms(&checked)
	if e := jaxson.CheckOperandWith(forms, dec(t, `{"$twice":3,"plus":1}`)); e != nil {
		t.Fatalf("registered form with its extra key rejected: %v", e)
	}
	if len(checked) != 1 {
		t.Fatalf("Check called %d times, want 1", len(checked))
	}
	if _, has := checked[0]["plus"]; !has {
		t.Fatalf("Check was not handed the whole form object: %v", checked[0])
	}
	// The form's own Check owns the rest of the validation.
	if e := jaxson.CheckOperandWith(forms, dec(t, `{"$twice":3,"bogus":1}`)); e == nil || e.Cat != "PROGRAM_ERROR" {
		t.Fatalf("form's own Check failure not surfaced: %v", e)
	}
}

func TestRegisteredFormIsFoundInsideAProgramAndInsideOtherOperands(t *testing.T) {
	prog := dec(t, `[
	  {"op":"set","path":["state","a"],"value":{"$twice":{"$compute":{"with":{},"expr":["add",1,2]}}}},
	  {"op":"set","path":["state","b"],"value":{"$compute":{"with":{"x":{"$twice":5}},"expr":["add",{"$v":"x"},1]}}}
	]`)
	if e := jaxson.CheckProgramForms(prog, jaxson.CoreInstructions(), nil, toyForms(nil)); e != nil {
		t.Fatalf("CheckProgramForms: %v", e)
	}
	if e := jaxson.CheckProgramForms(prog, jaxson.CoreInstructions(), nil, nil); e == nil {
		t.Fatalf("the same program passed with no forms registered")
	}
}

func TestRegisteredFormIsRecognisedInANestedScope(t *testing.T) {
	// A `for` body is checked with a cloned Checker; the clone must carry
	// the forms with it.
	prog := dec(t, `[
	  {"op":"for","in":[1,2],"as":"x","do":[
	    {"op":"set","path":["state","a"],"value":{"$twice":{"$path":["local","x"]}}}
	  ]}
	]`)
	if e := jaxson.CheckProgramForms(prog, jaxson.CoreInstructions(), nil, toyForms(nil)); e != nil {
		t.Fatalf("form inside a for body rejected: %v", e)
	}
}

func TestRegisteredFormEvaluates(t *testing.T) {
	state := map[string]any{}
	m := jaxson.NewMachine(nil, state, nil, 100, jaxson.CoreInstructions(), nil)
	if m.HasForm("$twice") {
		t.Fatalf("HasForm true before SetForms")
	}
	m.SetForms(toyForms(nil))
	if !m.HasForm("$twice") || m.HasForm("$thrice") {
		t.Fatalf("HasForm wrong after SetForms")
	}
	m.RunProgram(dec(t, `[
	  {"op":"set","path":["state","a"],"value":{"$twice":{"$compute":{"with":{},"expr":["add",1,2]}}}},
	  {"op":"set","path":["state","b"],"value":{"$twice":5,"plus":1}},
	  {"op":"set","path":["state","c"],"value":{"$compute":{"with":{"x":{"$twice":5}},"expr":["add",{"$v":"x"},1]}}}
	]`).([]any))
	if got := jaxson.Show(m.GetAt("state", []any{"a"})); got != "6" {
		t.Errorf("state.a = %s, want 6", got)
	}
	if got := jaxson.Show(m.GetAt("state", []any{"b"})); got != "11" {
		t.Errorf("state.b = %s, want 11", got)
	}
	if got := jaxson.Show(m.GetAt("state", []any{"c"})); got != "11" {
		t.Errorf("state.c = %s, want 11 (form inside a $compute binding)", got)
	}
	m.SetForms(nil)
	if m.HasForm("$twice") {
		t.Fatalf("HasForm true after SetForms(nil)")
	}
}

func TestFormEvalFailureIsTheMachinesFailure(t *testing.T) {
	m := jaxson.NewMachine(nil, map[string]any{}, nil, 100, jaxson.CoreInstructions(), nil)
	m.SetForms(toyForms(nil))
	var got *jaxson.Err
	func() {
		defer func() {
			if r := recover(); r != nil {
				got, _ = r.(*jaxson.Err)
			}
		}()
		m.RunProgram(dec(t, `[{"op":"set","path":["state","a"],"value":{"$twice":"x"}}]`).([]any))
	}()
	if got == nil || got.Cat != "EXECUTION_ERROR" || got.Code != "TYPE_ERROR" {
		t.Fatalf("got %v, want EXECUTION_ERROR/TYPE_ERROR", got)
	}
}
