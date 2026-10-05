// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the Apache License, Version 2.0.
// https://www.apache.org/licenses/LICENSE-2.0
package jaxson

// The package-level entry points: Run, plus the two pipeline stages a
// host package's own Run must repeat (contract-schema checking and
// value validation) because it cannot reach the unexported originals.

import "math/big"

// CheckSchema statically checks a contract schema (an inputSchema or
// outputSchema document) and returns a documented Err instead of
// panicking. Exported for host packages whose own Run repeats this
// stage of the pipeline.
func CheckSchema(s any) (err *Err) {
	defer func() {
		if r := recover(); r != nil {
			e, ok := r.(*Err)
			if !ok {
				panic(r)
			}
			err = e
		}
	}()
	checkSchema(s)
	return nil
}

// Validate returns "" when v satisfies the (already checked) contract
// schema s, otherwise the first failure message, with paths rooted at
// "$". The schema must have passed CheckSchema first.
func Validate(s, v any) string { return validate(s, Data(v), "$") }

// Run executes a package under the profile and returns either its output
// or a failure.
func (pr Profile) Run(p map[string]any) (out any, err *Err) {
	defer func() {
		if r := recover(); r != nil {
			e, ok := r.(*Err)
			if !ok {
				panic(r)
			}
			out, err = nil, e
		}
	}()
	if v, _ := p[pr.key()].(string); !pr.accepts(v) {
		fail("VERSION_ERROR", "", "unsupported or missing %s version", pr.key())
	}
	h := pr.begin(p)
	hasIn, hasOut := pr.hasSchema(p, "inputSchema"), pr.hasSchema(p, "outputSchema")
	if hasIn {
		checkSchema(p["inputSchema"])
	}
	if hasOut {
		checkSchema(p["outputSchema"])
	}
	h.Static(p)
	instructions := h.Instructions()
	c := NewChecker(instructions)
	c.Host = h.Host()
	forms := h.Forms()
	c.Forms = forms
	checkBlock(p["program"], c)
	limit := 100000
	if l, ok := p["limits"].(map[string]any); ok {
		for k := range l {
			if !pr.ownsLimit(k) {
				fail("VERSION_ERROR", "", "unsupported limit %q", k)
			}
		}
		if s, ok := l["steps"].(*big.Rat); ok && s.IsInt() && s.Sign() > 0 && s.Num().IsInt64() && s.Num().Int64() <= MaxCost {
			limit = int(s.Num().Int64())
		} else if _, has := l["steps"]; has {
			fail("VERSION_ERROR", "", "steps must be a positive integer no larger than 2^53-1")
		}
	}
	// The machine holds objects as *Object; the input arrives as maps unless
	// the reader already built it that way (ParsePackage), which Data accepts.
	input := Data(p["input"])
	if hasIn {
		if msg := validate(p["inputSchema"], input, "$"); msg != "" {
			fail("INPUT_ERROR", "", "%s", msg)
		}
	}
	// input is never written (no instruction may name it), and reads of it
	// are shared with the machine, so it needs no private copy.
	m := NewMachine(input, NewObject(0), nil, limit, instructions, CoreOperators())
	m.SetForms(forms)
	h.Start(m)
	h.AfterInput(m)
	m.RunProgram(p["program"].([]any))
	if hasOut {
		if msg := validate(p["outputSchema"], m.output, "$"); msg != "" {
			fail("OUTPUT_ERROR", "", "%s", msg)
		}
	}
	h.AfterOutput(m)
	return Legacy(m.output), nil
}
